package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	EmptyRpcParametersErr = errors.New("Rpc server method and path both required.")
	NilClientConnErr      = errors.New("Client conn is nil.")
	MethodNotImplErr      = errors.New("Rpc method not implemented.")
	NotImplProtoMsgErr    = errors.New("Not implment proto message.")
)

type (
	RPCRequest struct {
		ServicePath   string
		ServiceMethod string
		Metadata      metadata.MD
		Data          []byte
	}

	RPCResponse struct {
		DataChan   chan []byte
		Done       chan error
		RespHeader metadata.MD
		IsStream   bool
	}

	RPCClient struct {
		stub   grpcdynamic.Stub
		source *Source
	}
)

func NewRPCClient(ctx context.Context, conn *grpc.ClientConn) *RPCClient {
	return &RPCClient{
		stub:   grpcdynamic.NewStub(conn),
		source: NewSource(ctx, conn),
	}
}

func (g *RPCClient) Call(ctx context.Context, message RPCRequest) (*RPCResponse, error) {
	if message.ServiceMethod == "" || message.ServicePath == "" {
		return nil, EmptyRpcParametersErr
	}

	services, err := g.source.ListServices()
	if err != nil {
		return nil, err
	}

	var path string
	for _, v := range services {
		if strings.Contains(v, message.ServicePath) {
			path = v
			break
		}
	}

	if path == "" {
		return nil, fmt.Errorf("rpc server %s not implemented.", message.ServicePath)
	}

	desc, err := g.source.ResolveService(path)
	if err != nil {
		return nil, err
	}

	methodDesc := desc.FindMethodByName(message.ServiceMethod)
	if methodDesc == nil {
		return nil, fmt.Errorf("service path %s not include method %s", message.ServicePath, message.ServiceMethod)
	}

	ctx = metadata.NewOutgoingContext(ctx, message.Metadata)
	msg, err := g.createMsg(methodDesc, message.Data)
	if err != nil {
		return nil, err
	}

	var (
		headerMD  metadata.MD
		trailerMD metadata.MD

		done     = make(chan error, 2)
		dataChan = make(chan []byte, 100)
	)

	// invoke rpc, only support unary and service stream now
	if methodDesc.IsClientStreaming() || methodDesc.IsServerStreaming() {
		go g.invokeWithServiceStream(ctx, methodDesc, msg, done, dataChan)
	} else {
		go g.invokeWithUnary(ctx, methodDesc, msg, &headerMD, &trailerMD, done, dataChan)
	}

	return &RPCResponse{
		Done:       done,
		RespHeader: headerMD,
		DataChan:   dataChan,
		IsStream:   methodDesc.IsClientStreaming() || methodDesc.IsServerStreaming(),
	}, nil
}

func (g *RPCClient) createMsg(desc *desc.MethodDescriptor, data []byte) (*dynamic.Message, error) {
	msg := dynamic.NewMessage(desc.GetInputType())
	if err := msg.UnmarshalJSON(data); err != nil {
		return nil, err
	}

	return msg, nil
}

func (g *RPCClient) invokeWithUnary(ctx context.Context, methodDesc *desc.MethodDescriptor, msg *dynamic.Message, headerMD *metadata.MD, trailerMD *metadata.MD, done chan error, dataChan chan []byte) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	resp, err := g.stub.InvokeRpc(ctx, methodDesc, msg, grpc.Header(headerMD), grpc.Trailer(trailerMD))
	if err != nil {
		stat, ok := status.FromError(err)
		if !ok {
			done <- errors.New("unknown err")
			return
		}

		if stat.Code() == codes.Unavailable {
			done <- errors.New("rpc server unavailable")
			return
		}

		done <- err
		return
	}

	res, ok := resp.(*dynamic.Message)
	if !ok {
		done <- NotImplProtoMsgErr
		return
	}

	buf, err := res.MarshalJSON()
	if err != nil {
		done <- err
		return
	}

	dataChan <- buf
}

func (g *RPCClient) invokeWithServiceStream(ctx context.Context, methodDesc *desc.MethodDescriptor, msg *dynamic.Message, done chan error, dataChan chan []byte) {
	streamReq, err := g.stub.InvokeRpcServerStream(ctx, methodDesc, msg)
	if err != nil {
		done <- err
		return
	}

	for {
		resp, err := streamReq.RecvMsg()
		if err != nil {
			if err == io.EOF {
				close(dataChan)
				return
			}

			done <- err
			return
		}

		// respHeaders, err := streamReq.Header()
		// if err != nil {
		// 	doneChan <- err
		// 	return
		// }

		res, ok := resp.(*dynamic.Message)
		if !ok {
			done <- NotImplProtoMsgErr
			return
		}

		buf, err := res.MarshalJSON()
		if err != nil {
			done <- err
			return
		}

		dataChan <- buf

		resp.Reset()
	}
}
