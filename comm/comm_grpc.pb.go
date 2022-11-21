// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.15.8
// source: comm.proto

package comm

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// DataBaseClient is the client API for DataBase service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DataBaseClient interface {
	// Sends a sql
	SendSql(ctx context.Context, in *SqlRequest, opts ...grpc.CallOption) (*SqlResult, error)
	Scan(ctx context.Context, in *SqlRequest, opts ...grpc.CallOption) (*SqlResult, error)
}

type dataBaseClient struct {
	cc grpc.ClientConnInterface
}

func NewDataBaseClient(cc grpc.ClientConnInterface) DataBaseClient {
	return &dataBaseClient{cc}
}

func (c *dataBaseClient) SendSql(ctx context.Context, in *SqlRequest, opts ...grpc.CallOption) (*SqlResult, error) {
	out := new(SqlResult)
	err := c.cc.Invoke(ctx, "/comm.DataBase/SendSql", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dataBaseClient) Scan(ctx context.Context, in *SqlRequest, opts ...grpc.CallOption) (*SqlResult, error) {
	out := new(SqlResult)
	err := c.cc.Invoke(ctx, "/comm.DataBase/Scan", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DataBaseServer is the server API for DataBase service.
// All implementations must embed UnimplementedDataBaseServer
// for forward compatibility
type DataBaseServer interface {
	// Sends a sql
	SendSql(context.Context, *SqlRequest) (*SqlResult, error)
	Scan(context.Context, *SqlRequest) (*SqlResult, error)
	mustEmbedUnimplementedDataBaseServer()
}

// UnimplementedDataBaseServer must be embedded to have forward compatible implementations.
type UnimplementedDataBaseServer struct {
}

func (UnimplementedDataBaseServer) SendSql(context.Context, *SqlRequest) (*SqlResult, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SendSql not implemented")
}
func (UnimplementedDataBaseServer) Scan(context.Context, *SqlRequest) (*SqlResult, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Scan not implemented")
}
func (UnimplementedDataBaseServer) mustEmbedUnimplementedDataBaseServer() {}

// UnsafeDataBaseServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DataBaseServer will
// result in compilation errors.
type UnsafeDataBaseServer interface {
	mustEmbedUnimplementedDataBaseServer()
}

func RegisterDataBaseServer(s grpc.ServiceRegistrar, srv DataBaseServer) {
	s.RegisterService(&DataBase_ServiceDesc, srv)
}

func _DataBase_SendSql_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SqlRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DataBaseServer).SendSql(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/comm.DataBase/SendSql",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DataBaseServer).SendSql(ctx, req.(*SqlRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DataBase_Scan_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SqlRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DataBaseServer).Scan(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/comm.DataBase/Scan",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DataBaseServer).Scan(ctx, req.(*SqlRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DataBase_ServiceDesc is the grpc.ServiceDesc for DataBase service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DataBase_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "comm.DataBase",
	HandlerType: (*DataBaseServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SendSql",
			Handler:    _DataBase_SendSql_Handler,
		},
		{
			MethodName: "Scan",
			Handler:    _DataBase_Scan_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "comm.proto",
}