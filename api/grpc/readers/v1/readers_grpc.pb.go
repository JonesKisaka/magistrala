// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v6.30.2
// source: readers/v1/readers.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	ReadersService_ReadMessages_FullMethodName = "/readers.v1.ReadersService/ReadMessages"
)

// ReadersServiceClient is the client API for ReadersService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// ReadersService is a service that provides access to
// readers functionalities for Magistrala services.
type ReadersServiceClient interface {
	ReadMessages(ctx context.Context, in *ReadMessagesReq, opts ...grpc.CallOption) (*ReadMessagesRes, error)
}

type readersServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewReadersServiceClient(cc grpc.ClientConnInterface) ReadersServiceClient {
	return &readersServiceClient{cc}
}

func (c *readersServiceClient) ReadMessages(ctx context.Context, in *ReadMessagesReq, opts ...grpc.CallOption) (*ReadMessagesRes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ReadMessagesRes)
	err := c.cc.Invoke(ctx, ReadersService_ReadMessages_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ReadersServiceServer is the server API for ReadersService service.
// All implementations must embed UnimplementedReadersServiceServer
// for forward compatibility.
//
// ReadersService is a service that provides access to
// readers functionalities for Magistrala services.
type ReadersServiceServer interface {
	ReadMessages(context.Context, *ReadMessagesReq) (*ReadMessagesRes, error)
	mustEmbedUnimplementedReadersServiceServer()
}

// UnimplementedReadersServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedReadersServiceServer struct{}

func (UnimplementedReadersServiceServer) ReadMessages(context.Context, *ReadMessagesReq) (*ReadMessagesRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReadMessages not implemented")
}
func (UnimplementedReadersServiceServer) mustEmbedUnimplementedReadersServiceServer() {}
func (UnimplementedReadersServiceServer) testEmbeddedByValue()                        {}

// UnsafeReadersServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ReadersServiceServer will
// result in compilation errors.
type UnsafeReadersServiceServer interface {
	mustEmbedUnimplementedReadersServiceServer()
}

func RegisterReadersServiceServer(s grpc.ServiceRegistrar, srv ReadersServiceServer) {
	// If the following call pancis, it indicates UnimplementedReadersServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&ReadersService_ServiceDesc, srv)
}

func _ReadersService_ReadMessages_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReadMessagesReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ReadersServiceServer).ReadMessages(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ReadersService_ReadMessages_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ReadersServiceServer).ReadMessages(ctx, req.(*ReadMessagesReq))
	}
	return interceptor(ctx, in, info, handler)
}

// ReadersService_ServiceDesc is the grpc.ServiceDesc for ReadersService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ReadersService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "readers.v1.ReadersService",
	HandlerType: (*ReadersServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ReadMessages",
			Handler:    _ReadersService_ReadMessages_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "readers/v1/readers.proto",
}
