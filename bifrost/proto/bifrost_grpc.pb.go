// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.5
// source: bifrost/proto/bifrost.proto

package proto

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

// BifrostClient is the client API for Bifrost service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BifrostClient interface {
	CommandStart(ctx context.Context, in *CommandStartRequest, opts ...grpc.CallOption) (*CommandStartResponse, error)
	CommandEnd(ctx context.Context, in *CommandEndRequest, opts ...grpc.CallOption) (*CommandEndResponse, error)
	ListCommands(ctx context.Context, in *ListCommandsRequest, opts ...grpc.CallOption) (*ListCommandsResponse, error)
	WaitForCommand(ctx context.Context, in *WaitForCommandRequest, opts ...grpc.CallOption) (*WaitForCommandResponse, error)
	CacheCommand(ctx context.Context, in *CacheCommandRequest, opts ...grpc.CallOption) (*CacheCommandResponse, error)
}

type bifrostClient struct {
	cc grpc.ClientConnInterface
}

func NewBifrostClient(cc grpc.ClientConnInterface) BifrostClient {
	return &bifrostClient{cc}
}

func (c *bifrostClient) CommandStart(ctx context.Context, in *CommandStartRequest, opts ...grpc.CallOption) (*CommandStartResponse, error) {
	out := new(CommandStartResponse)
	err := c.cc.Invoke(ctx, "/Bifrost/CommandStart", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bifrostClient) CommandEnd(ctx context.Context, in *CommandEndRequest, opts ...grpc.CallOption) (*CommandEndResponse, error) {
	out := new(CommandEndResponse)
	err := c.cc.Invoke(ctx, "/Bifrost/CommandEnd", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bifrostClient) ListCommands(ctx context.Context, in *ListCommandsRequest, opts ...grpc.CallOption) (*ListCommandsResponse, error) {
	out := new(ListCommandsResponse)
	err := c.cc.Invoke(ctx, "/Bifrost/ListCommands", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bifrostClient) WaitForCommand(ctx context.Context, in *WaitForCommandRequest, opts ...grpc.CallOption) (*WaitForCommandResponse, error) {
	out := new(WaitForCommandResponse)
	err := c.cc.Invoke(ctx, "/Bifrost/WaitForCommand", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bifrostClient) CacheCommand(ctx context.Context, in *CacheCommandRequest, opts ...grpc.CallOption) (*CacheCommandResponse, error) {
	out := new(CacheCommandResponse)
	err := c.cc.Invoke(ctx, "/Bifrost/CacheCommand", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BifrostServer is the server API for Bifrost service.
// All implementations must embed UnimplementedBifrostServer
// for forward compatibility
type BifrostServer interface {
	CommandStart(context.Context, *CommandStartRequest) (*CommandStartResponse, error)
	CommandEnd(context.Context, *CommandEndRequest) (*CommandEndResponse, error)
	ListCommands(context.Context, *ListCommandsRequest) (*ListCommandsResponse, error)
	WaitForCommand(context.Context, *WaitForCommandRequest) (*WaitForCommandResponse, error)
	CacheCommand(context.Context, *CacheCommandRequest) (*CacheCommandResponse, error)
	mustEmbedUnimplementedBifrostServer()
}

// UnimplementedBifrostServer must be embedded to have forward compatible implementations.
type UnimplementedBifrostServer struct {
}

func (UnimplementedBifrostServer) CommandStart(context.Context, *CommandStartRequest) (*CommandStartResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CommandStart not implemented")
}
func (UnimplementedBifrostServer) CommandEnd(context.Context, *CommandEndRequest) (*CommandEndResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CommandEnd not implemented")
}
func (UnimplementedBifrostServer) ListCommands(context.Context, *ListCommandsRequest) (*ListCommandsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListCommands not implemented")
}
func (UnimplementedBifrostServer) WaitForCommand(context.Context, *WaitForCommandRequest) (*WaitForCommandResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WaitForCommand not implemented")
}
func (UnimplementedBifrostServer) CacheCommand(context.Context, *CacheCommandRequest) (*CacheCommandResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CacheCommand not implemented")
}
func (UnimplementedBifrostServer) mustEmbedUnimplementedBifrostServer() {}

// UnsafeBifrostServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BifrostServer will
// result in compilation errors.
type UnsafeBifrostServer interface {
	mustEmbedUnimplementedBifrostServer()
}

func RegisterBifrostServer(s grpc.ServiceRegistrar, srv BifrostServer) {
	s.RegisterService(&Bifrost_ServiceDesc, srv)
}

func _Bifrost_CommandStart_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CommandStartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BifrostServer).CommandStart(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Bifrost/CommandStart",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BifrostServer).CommandStart(ctx, req.(*CommandStartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Bifrost_CommandEnd_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CommandEndRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BifrostServer).CommandEnd(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Bifrost/CommandEnd",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BifrostServer).CommandEnd(ctx, req.(*CommandEndRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Bifrost_ListCommands_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListCommandsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BifrostServer).ListCommands(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Bifrost/ListCommands",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BifrostServer).ListCommands(ctx, req.(*ListCommandsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Bifrost_WaitForCommand_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WaitForCommandRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BifrostServer).WaitForCommand(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Bifrost/WaitForCommand",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BifrostServer).WaitForCommand(ctx, req.(*WaitForCommandRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Bifrost_CacheCommand_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CacheCommandRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BifrostServer).CacheCommand(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Bifrost/CacheCommand",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BifrostServer).CacheCommand(ctx, req.(*CacheCommandRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Bifrost_ServiceDesc is the grpc.ServiceDesc for Bifrost service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Bifrost_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "Bifrost",
	HandlerType: (*BifrostServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CommandStart",
			Handler:    _Bifrost_CommandStart_Handler,
		},
		{
			MethodName: "CommandEnd",
			Handler:    _Bifrost_CommandEnd_Handler,
		},
		{
			MethodName: "ListCommands",
			Handler:    _Bifrost_ListCommands_Handler,
		},
		{
			MethodName: "WaitForCommand",
			Handler:    _Bifrost_WaitForCommand_Handler,
		},
		{
			MethodName: "CacheCommand",
			Handler:    _Bifrost_CacheCommand_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "bifrost/proto/bifrost.proto",
}
