package transform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	tx "github.com/KKKKjl/tinykit/internal/context"
	"github.com/KKKKjl/tinykit/internal/request"
	"github.com/KKKKjl/tinykit/logger"
	"google.golang.org/grpc/metadata"
)

var (
	log     = logger.GetLogger()
	mainLog = log.WithField("prefix", "transform")
)

var (
	RpcSerializeType = "X-RPC-SerializeType"
	RpcServicePath   = "X-RPC-ServicePath"
	RpcServiceMethod = "X-RPC-ServiceMethod"
	RpcPrefix        = "X-RPC-Metadata-"
)

var (
	NotImplProtoMessageError = errors.New("Not implment proto message.")
	EmptyServiceMethodErr    = errors.New("Service method is required.")
	EmptyServicePathErr      = errors.New("Service path is required.")
	GetRpcMetaDataErr        = errors.New("Fail to get rpc metadata from ctx.")
)

type (
	ApiDefinition struct {
	}

	ApiDefinitionParser struct {
	}

	RPCMetadata struct {
		ContentType string
	}
)

// convert rpc request to http request
func (a *ApiDefinitionParser) TransformToApi(ctx context.Context, w http.ResponseWriter, r *http.Request, reWrite bool) error {
	meta, ok := a.GetMetaDataFromCtx(ctx)
	if !ok {
		return GetRpcMetaDataErr
	}

	if reWrite {
		w.Header().Set("Content-Type", meta.ContentType)
	}

	var buf []byte
	var err error
	if buf, err = json.Marshal(meta); err != nil {
		return err
	}

	if _, err := w.Write(buf); err != nil {
		return err
	}

	return nil
}

// convert http request to rpc request
func (a *ApiDefinitionParser) TransformToRPC(ctx tx.HttpContext) (request.RPCRequest, error) {
	var (
		msg              request.RPCRequest
		rpcServicePath   string
		rpcServiceMethod string
	)

	buf, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		return msg, err
	}
	defer ctx.Request.Body.Close()

	if len(buf) == 0 {
		buf = []byte("{}")
	}

	rpcServicePath = ctx.Request.Header.Get(RpcServicePath)
	if rpcServicePath == "" {
		return msg, EmptyServicePathErr
	}

	rpcServiceMethod = ctx.Request.Header.Get(RpcServiceMethod)
	if rpcServiceMethod == "" {
		return msg, EmptyServiceMethodErr
	}

	msg.ServicePath = rpcServicePath
	msg.ServiceMethod = rpcServiceMethod
	msg.Data = buf
	msg.Metadata = a.GetMetaDataFromHeaders(ctx.Request.Header.Clone())

	return msg, nil
}

func (a *ApiDefinitionParser) ruleMatcher(w http.ResponseWriter, r *http.Request) error {
	panic("impl me")
}

func (a *ApiDefinitionParser) GetMetaDataFromCtx(ctx context.Context) (meta *RPCMetadata, ok bool) {
	meta, ok = ctx.Value(ApiDefinition{}).(*RPCMetadata)
	return
}

func (a *ApiDefinitionParser) GetMetaDataFromHeaders(headers http.Header) metadata.MD {
	md := make(map[string]string)

	for k, v := range headers {
		if strings.HasPrefix(k, RpcPrefix) {
			k = strings.TrimPrefix(k, RpcPrefix)
			md[strings.ToLower(k)] = v[0]
		}
	}

	return metadata.New(md)
}

func (a *ApiDefinitionParser) ParseDefinition(r io.Reader) (definition *ApiDefinition) {
	if err := json.NewDecoder(r).Decode(&definition); err != nil {
		mainLog.Errorf("parse api definition error: %s", err.Error())
	}

	return
}

// Check if protocol conversion is required.
func (a *ApiDefinitionParser) IsMatchTransformRule(ctx tx.HttpContext) bool {
	serializeType := strings.ToLower(ctx.Request.Header.Get(RpcSerializeType))
	return serializeType == "protobuf" || serializeType == "json"
}

func (a *ApiDefinitionParser) ToHeaders(md metadata.MD) map[string]string {
	headers := make(map[string]string, md.Len())

	for k, v := range md {
		headers[RpcPrefix+strings.ToUpper(k)] = v[0]
	}
	return headers
}
