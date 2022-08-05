package request

import (
	"context"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

type Source struct {
	client *grpcreflect.Client
}

func NewSource(ctx context.Context, conn *grpc.ClientConn) *Source {
	return &Source{
		client: grpcreflect.NewClient(ctx, grpc_reflection_v1alpha.NewServerReflectionClient(conn)),
	}
}

func (s *Source) ListServices() ([]string, error) {
	return s.client.ListServices()
}

func (s *Source) ResolveService(name string) (*desc.ServiceDescriptor, error) {
	return s.client.ResolveService(name)
}
