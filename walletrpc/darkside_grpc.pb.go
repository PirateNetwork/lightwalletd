// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.19.4
// source: darkside.proto

package walletrpc

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

// DarksideStreamerClient is the client API for DarksideStreamer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DarksideStreamerClient interface {
	// Reset reverts all darksidewalletd state (active block range, latest height,
	// staged blocks and transactions) and lightwalletd state (cache) to empty,
	// the same as the initial state. This occurs synchronously and instantaneously;
	// no reorg happens in lightwalletd. This is good to do before each independent
	// test so that no state leaks from one test to another.
	// Also sets (some of) the values returned by GetLightdInfo(). The Sapling
	// activation height specified here must be where the block range starts.
	Reset(ctx context.Context, in *DarksideMetaState, opts ...grpc.CallOption) (*Empty, error)
	// StageBlocksStream accepts a list of blocks and saves them into the blocks
	// staging area until ApplyStaged() is called; there is no immediate effect on
	// the mock zcashd. Blocks are hex-encoded. Order is important, see ApplyStaged.
	StageBlocksStream(ctx context.Context, opts ...grpc.CallOption) (DarksideStreamer_StageBlocksStreamClient, error)
	// StageBlocks is the same as StageBlocksStream() except the blocks are fetched
	// from the given URL. Blocks are one per line, hex-encoded (not JSON).
	StageBlocks(ctx context.Context, in *DarksideBlocksURL, opts ...grpc.CallOption) (*Empty, error)
	// StageBlocksCreate is like the previous two, except it creates 'count'
	// empty blocks at consecutive heights starting at height 'height'. The
	// 'nonce' is part of the header, so it contributes to the block hash; this
	// lets you create identical blocks (same transactions and height), but with
	// different hashes.
	StageBlocksCreate(ctx context.Context, in *DarksideEmptyBlocks, opts ...grpc.CallOption) (*Empty, error)
	// StageTransactionsStream stores the given transaction-height pairs in the
	// staging area until ApplyStaged() is called. Note that these transactions
	// are not returned by the production GetTransaction() gRPC until they
	// appear in a "mined" block (contained in the active blockchain presented
	// by the mock zcashd).
	StageTransactionsStream(ctx context.Context, opts ...grpc.CallOption) (DarksideStreamer_StageTransactionsStreamClient, error)
	// StageTransactions is the same except the transactions are fetched from
	// the given url. They are all staged into the block at the given height.
	// Staging transactions to different heights requires multiple calls.
	StageTransactions(ctx context.Context, in *DarksideTransactionsURL, opts ...grpc.CallOption) (*Empty, error)
	// ApplyStaged iterates the list of blocks that were staged by the
	// StageBlocks*() gRPCs, in the order they were staged, and "merges" each
	// into the active, working blocks list that the mock zcashd is presenting
	// to lightwalletd. Even as each block is applied, the active list can't
	// have gaps; if the active block range is 1000-1006, and the staged block
	// range is 1003-1004, the resulting range is 1000-1004, with 1000-1002
	// unchanged, blocks 1003-1004 from the new range, and 1005-1006 dropped.
	//
	// After merging all blocks, ApplyStaged() appends staged transactions (in
	// the order received) into each one's corresponding (by height) block
	// The staging area is then cleared.
	//
	// The argument specifies the latest block height that mock zcashd reports
	// (i.e. what's returned by GetLatestBlock). Note that ApplyStaged() can
	// also be used to simply advance the latest block height presented by mock
	// zcashd. That is, there doesn't need to be anything in the staging area.
	ApplyStaged(ctx context.Context, in *DarksideHeight, opts ...grpc.CallOption) (*Empty, error)
	// Calls to the production gRPC SendTransaction() store the transaction in
	// a separate area (not the staging area); this method returns all transactions
	// in this separate area, which is then cleared. The height returned
	// with each transaction is -1 (invalid) since these transactions haven't
	// been mined yet. The intention is that the transactions returned here can
	// then, for example, be given to StageTransactions() to get them "mined"
	// into a specified block on the next ApplyStaged().
	GetIncomingTransactions(ctx context.Context, in *Empty, opts ...grpc.CallOption) (DarksideStreamer_GetIncomingTransactionsClient, error)
	// Clear the incoming transaction pool.
	ClearIncomingTransactions(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Empty, error)
	// Add a GetAddressUtxosReply entry to be returned by GetAddressUtxos().
	// There is no staging or applying for these, very simple.
	AddAddressUtxo(ctx context.Context, in *GetAddressUtxosReply, opts ...grpc.CallOption) (*Empty, error)
	// Clear the list of GetAddressUtxos entries (can't fail)
	ClearAddressUtxo(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Empty, error)
}

type darksideStreamerClient struct {
	cc grpc.ClientConnInterface
}

func NewDarksideStreamerClient(cc grpc.ClientConnInterface) DarksideStreamerClient {
	return &darksideStreamerClient{cc}
}

func (c *darksideStreamerClient) Reset(ctx context.Context, in *DarksideMetaState, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/pirate.wallet.sdk.rpc.DarksideStreamer/Reset", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *darksideStreamerClient) StageBlocksStream(ctx context.Context, opts ...grpc.CallOption) (DarksideStreamer_StageBlocksStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &DarksideStreamer_ServiceDesc.Streams[0], "/pirate.wallet.sdk.rpc.DarksideStreamer/StageBlocksStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &darksideStreamerStageBlocksStreamClient{stream}
	return x, nil
}

type DarksideStreamer_StageBlocksStreamClient interface {
	Send(*DarksideBlock) error
	CloseAndRecv() (*Empty, error)
	grpc.ClientStream
}

type darksideStreamerStageBlocksStreamClient struct {
	grpc.ClientStream
}

func (x *darksideStreamerStageBlocksStreamClient) Send(m *DarksideBlock) error {
	return x.ClientStream.SendMsg(m)
}

func (x *darksideStreamerStageBlocksStreamClient) CloseAndRecv() (*Empty, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(Empty)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *darksideStreamerClient) StageBlocks(ctx context.Context, in *DarksideBlocksURL, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/pirate.wallet.sdk.rpc.DarksideStreamer/StageBlocks", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *darksideStreamerClient) StageBlocksCreate(ctx context.Context, in *DarksideEmptyBlocks, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/pirate.wallet.sdk.rpc.DarksideStreamer/StageBlocksCreate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *darksideStreamerClient) StageTransactionsStream(ctx context.Context, opts ...grpc.CallOption) (DarksideStreamer_StageTransactionsStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &DarksideStreamer_ServiceDesc.Streams[1], "/pirate.wallet.sdk.rpc.DarksideStreamer/StageTransactionsStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &darksideStreamerStageTransactionsStreamClient{stream}
	return x, nil
}

type DarksideStreamer_StageTransactionsStreamClient interface {
	Send(*RawTransaction) error
	CloseAndRecv() (*Empty, error)
	grpc.ClientStream
}

type darksideStreamerStageTransactionsStreamClient struct {
	grpc.ClientStream
}

func (x *darksideStreamerStageTransactionsStreamClient) Send(m *RawTransaction) error {
	return x.ClientStream.SendMsg(m)
}

func (x *darksideStreamerStageTransactionsStreamClient) CloseAndRecv() (*Empty, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(Empty)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *darksideStreamerClient) StageTransactions(ctx context.Context, in *DarksideTransactionsURL, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/pirate.wallet.sdk.rpc.DarksideStreamer/StageTransactions", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *darksideStreamerClient) ApplyStaged(ctx context.Context, in *DarksideHeight, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/pirate.wallet.sdk.rpc.DarksideStreamer/ApplyStaged", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *darksideStreamerClient) GetIncomingTransactions(ctx context.Context, in *Empty, opts ...grpc.CallOption) (DarksideStreamer_GetIncomingTransactionsClient, error) {
	stream, err := c.cc.NewStream(ctx, &DarksideStreamer_ServiceDesc.Streams[2], "/pirate.wallet.sdk.rpc.DarksideStreamer/GetIncomingTransactions", opts...)
	if err != nil {
		return nil, err
	}
	x := &darksideStreamerGetIncomingTransactionsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type DarksideStreamer_GetIncomingTransactionsClient interface {
	Recv() (*RawTransaction, error)
	grpc.ClientStream
}

type darksideStreamerGetIncomingTransactionsClient struct {
	grpc.ClientStream
}

func (x *darksideStreamerGetIncomingTransactionsClient) Recv() (*RawTransaction, error) {
	m := new(RawTransaction)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *darksideStreamerClient) ClearIncomingTransactions(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/pirate.wallet.sdk.rpc.DarksideStreamer/ClearIncomingTransactions", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *darksideStreamerClient) AddAddressUtxo(ctx context.Context, in *GetAddressUtxosReply, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/pirate.wallet.sdk.rpc.DarksideStreamer/AddAddressUtxo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *darksideStreamerClient) ClearAddressUtxo(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/pirate.wallet.sdk.rpc.DarksideStreamer/ClearAddressUtxo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DarksideStreamerServer is the server API for DarksideStreamer service.
// All implementations must embed UnimplementedDarksideStreamerServer
// for forward compatibility
type DarksideStreamerServer interface {
	// Reset reverts all darksidewalletd state (active block range, latest height,
	// staged blocks and transactions) and lightwalletd state (cache) to empty,
	// the same as the initial state. This occurs synchronously and instantaneously;
	// no reorg happens in lightwalletd. This is good to do before each independent
	// test so that no state leaks from one test to another.
	// Also sets (some of) the values returned by GetLightdInfo(). The Sapling
	// activation height specified here must be where the block range starts.
	Reset(context.Context, *DarksideMetaState) (*Empty, error)
	// StageBlocksStream accepts a list of blocks and saves them into the blocks
	// staging area until ApplyStaged() is called; there is no immediate effect on
	// the mock zcashd. Blocks are hex-encoded. Order is important, see ApplyStaged.
	StageBlocksStream(DarksideStreamer_StageBlocksStreamServer) error
	// StageBlocks is the same as StageBlocksStream() except the blocks are fetched
	// from the given URL. Blocks are one per line, hex-encoded (not JSON).
	StageBlocks(context.Context, *DarksideBlocksURL) (*Empty, error)
	// StageBlocksCreate is like the previous two, except it creates 'count'
	// empty blocks at consecutive heights starting at height 'height'. The
	// 'nonce' is part of the header, so it contributes to the block hash; this
	// lets you create identical blocks (same transactions and height), but with
	// different hashes.
	StageBlocksCreate(context.Context, *DarksideEmptyBlocks) (*Empty, error)
	// StageTransactionsStream stores the given transaction-height pairs in the
	// staging area until ApplyStaged() is called. Note that these transactions
	// are not returned by the production GetTransaction() gRPC until they
	// appear in a "mined" block (contained in the active blockchain presented
	// by the mock zcashd).
	StageTransactionsStream(DarksideStreamer_StageTransactionsStreamServer) error
	// StageTransactions is the same except the transactions are fetched from
	// the given url. They are all staged into the block at the given height.
	// Staging transactions to different heights requires multiple calls.
	StageTransactions(context.Context, *DarksideTransactionsURL) (*Empty, error)
	// ApplyStaged iterates the list of blocks that were staged by the
	// StageBlocks*() gRPCs, in the order they were staged, and "merges" each
	// into the active, working blocks list that the mock zcashd is presenting
	// to lightwalletd. Even as each block is applied, the active list can't
	// have gaps; if the active block range is 1000-1006, and the staged block
	// range is 1003-1004, the resulting range is 1000-1004, with 1000-1002
	// unchanged, blocks 1003-1004 from the new range, and 1005-1006 dropped.
	//
	// After merging all blocks, ApplyStaged() appends staged transactions (in
	// the order received) into each one's corresponding (by height) block
	// The staging area is then cleared.
	//
	// The argument specifies the latest block height that mock zcashd reports
	// (i.e. what's returned by GetLatestBlock). Note that ApplyStaged() can
	// also be used to simply advance the latest block height presented by mock
	// zcashd. That is, there doesn't need to be anything in the staging area.
	ApplyStaged(context.Context, *DarksideHeight) (*Empty, error)
	// Calls to the production gRPC SendTransaction() store the transaction in
	// a separate area (not the staging area); this method returns all transactions
	// in this separate area, which is then cleared. The height returned
	// with each transaction is -1 (invalid) since these transactions haven't
	// been mined yet. The intention is that the transactions returned here can
	// then, for example, be given to StageTransactions() to get them "mined"
	// into a specified block on the next ApplyStaged().
	GetIncomingTransactions(*Empty, DarksideStreamer_GetIncomingTransactionsServer) error
	// Clear the incoming transaction pool.
	ClearIncomingTransactions(context.Context, *Empty) (*Empty, error)
	// Add a GetAddressUtxosReply entry to be returned by GetAddressUtxos().
	// There is no staging or applying for these, very simple.
	AddAddressUtxo(context.Context, *GetAddressUtxosReply) (*Empty, error)
	// Clear the list of GetAddressUtxos entries (can't fail)
	ClearAddressUtxo(context.Context, *Empty) (*Empty, error)
	mustEmbedUnimplementedDarksideStreamerServer()
}

// UnimplementedDarksideStreamerServer must be embedded to have forward compatible implementations.
type UnimplementedDarksideStreamerServer struct {
}

func (UnimplementedDarksideStreamerServer) Reset(context.Context, *DarksideMetaState) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Reset not implemented")
}
func (UnimplementedDarksideStreamerServer) StageBlocksStream(DarksideStreamer_StageBlocksStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method StageBlocksStream not implemented")
}
func (UnimplementedDarksideStreamerServer) StageBlocks(context.Context, *DarksideBlocksURL) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StageBlocks not implemented")
}
func (UnimplementedDarksideStreamerServer) StageBlocksCreate(context.Context, *DarksideEmptyBlocks) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StageBlocksCreate not implemented")
}
func (UnimplementedDarksideStreamerServer) StageTransactionsStream(DarksideStreamer_StageTransactionsStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method StageTransactionsStream not implemented")
}
func (UnimplementedDarksideStreamerServer) StageTransactions(context.Context, *DarksideTransactionsURL) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StageTransactions not implemented")
}
func (UnimplementedDarksideStreamerServer) ApplyStaged(context.Context, *DarksideHeight) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ApplyStaged not implemented")
}
func (UnimplementedDarksideStreamerServer) GetIncomingTransactions(*Empty, DarksideStreamer_GetIncomingTransactionsServer) error {
	return status.Errorf(codes.Unimplemented, "method GetIncomingTransactions not implemented")
}
func (UnimplementedDarksideStreamerServer) ClearIncomingTransactions(context.Context, *Empty) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ClearIncomingTransactions not implemented")
}
func (UnimplementedDarksideStreamerServer) AddAddressUtxo(context.Context, *GetAddressUtxosReply) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddAddressUtxo not implemented")
}
func (UnimplementedDarksideStreamerServer) ClearAddressUtxo(context.Context, *Empty) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ClearAddressUtxo not implemented")
}
func (UnimplementedDarksideStreamerServer) mustEmbedUnimplementedDarksideStreamerServer() {}

// UnsafeDarksideStreamerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DarksideStreamerServer will
// result in compilation errors.
type UnsafeDarksideStreamerServer interface {
	mustEmbedUnimplementedDarksideStreamerServer()
}

func RegisterDarksideStreamerServer(s grpc.ServiceRegistrar, srv DarksideStreamerServer) {
	s.RegisterService(&DarksideStreamer_ServiceDesc, srv)
}

func _DarksideStreamer_Reset_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DarksideMetaState)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DarksideStreamerServer).Reset(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pirate.wallet.sdk.rpc.DarksideStreamer/Reset",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DarksideStreamerServer).Reset(ctx, req.(*DarksideMetaState))
	}
	return interceptor(ctx, in, info, handler)
}

func _DarksideStreamer_StageBlocksStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(DarksideStreamerServer).StageBlocksStream(&darksideStreamerStageBlocksStreamServer{stream})
}

type DarksideStreamer_StageBlocksStreamServer interface {
	SendAndClose(*Empty) error
	Recv() (*DarksideBlock, error)
	grpc.ServerStream
}

type darksideStreamerStageBlocksStreamServer struct {
	grpc.ServerStream
}

func (x *darksideStreamerStageBlocksStreamServer) SendAndClose(m *Empty) error {
	return x.ServerStream.SendMsg(m)
}

func (x *darksideStreamerStageBlocksStreamServer) Recv() (*DarksideBlock, error) {
	m := new(DarksideBlock)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _DarksideStreamer_StageBlocks_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DarksideBlocksURL)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DarksideStreamerServer).StageBlocks(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pirate.wallet.sdk.rpc.DarksideStreamer/StageBlocks",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DarksideStreamerServer).StageBlocks(ctx, req.(*DarksideBlocksURL))
	}
	return interceptor(ctx, in, info, handler)
}

func _DarksideStreamer_StageBlocksCreate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DarksideEmptyBlocks)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DarksideStreamerServer).StageBlocksCreate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pirate.wallet.sdk.rpc.DarksideStreamer/StageBlocksCreate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DarksideStreamerServer).StageBlocksCreate(ctx, req.(*DarksideEmptyBlocks))
	}
	return interceptor(ctx, in, info, handler)
}

func _DarksideStreamer_StageTransactionsStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(DarksideStreamerServer).StageTransactionsStream(&darksideStreamerStageTransactionsStreamServer{stream})
}

type DarksideStreamer_StageTransactionsStreamServer interface {
	SendAndClose(*Empty) error
	Recv() (*RawTransaction, error)
	grpc.ServerStream
}

type darksideStreamerStageTransactionsStreamServer struct {
	grpc.ServerStream
}

func (x *darksideStreamerStageTransactionsStreamServer) SendAndClose(m *Empty) error {
	return x.ServerStream.SendMsg(m)
}

func (x *darksideStreamerStageTransactionsStreamServer) Recv() (*RawTransaction, error) {
	m := new(RawTransaction)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _DarksideStreamer_StageTransactions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DarksideTransactionsURL)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DarksideStreamerServer).StageTransactions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pirate.wallet.sdk.rpc.DarksideStreamer/StageTransactions",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DarksideStreamerServer).StageTransactions(ctx, req.(*DarksideTransactionsURL))
	}
	return interceptor(ctx, in, info, handler)
}

func _DarksideStreamer_ApplyStaged_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DarksideHeight)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DarksideStreamerServer).ApplyStaged(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pirate.wallet.sdk.rpc.DarksideStreamer/ApplyStaged",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DarksideStreamerServer).ApplyStaged(ctx, req.(*DarksideHeight))
	}
	return interceptor(ctx, in, info, handler)
}

func _DarksideStreamer_GetIncomingTransactions_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(DarksideStreamerServer).GetIncomingTransactions(m, &darksideStreamerGetIncomingTransactionsServer{stream})
}

type DarksideStreamer_GetIncomingTransactionsServer interface {
	Send(*RawTransaction) error
	grpc.ServerStream
}

type darksideStreamerGetIncomingTransactionsServer struct {
	grpc.ServerStream
}

func (x *darksideStreamerGetIncomingTransactionsServer) Send(m *RawTransaction) error {
	return x.ServerStream.SendMsg(m)
}

func _DarksideStreamer_ClearIncomingTransactions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DarksideStreamerServer).ClearIncomingTransactions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pirate.wallet.sdk.rpc.DarksideStreamer/ClearIncomingTransactions",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DarksideStreamerServer).ClearIncomingTransactions(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _DarksideStreamer_AddAddressUtxo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetAddressUtxosReply)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DarksideStreamerServer).AddAddressUtxo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pirate.wallet.sdk.rpc.DarksideStreamer/AddAddressUtxo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DarksideStreamerServer).AddAddressUtxo(ctx, req.(*GetAddressUtxosReply))
	}
	return interceptor(ctx, in, info, handler)
}

func _DarksideStreamer_ClearAddressUtxo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DarksideStreamerServer).ClearAddressUtxo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pirate.wallet.sdk.rpc.DarksideStreamer/ClearAddressUtxo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DarksideStreamerServer).ClearAddressUtxo(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// DarksideStreamer_ServiceDesc is the grpc.ServiceDesc for DarksideStreamer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DarksideStreamer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "pirate.wallet.sdk.rpc.DarksideStreamer",
	HandlerType: (*DarksideStreamerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Reset",
			Handler:    _DarksideStreamer_Reset_Handler,
		},
		{
			MethodName: "StageBlocks",
			Handler:    _DarksideStreamer_StageBlocks_Handler,
		},
		{
			MethodName: "StageBlocksCreate",
			Handler:    _DarksideStreamer_StageBlocksCreate_Handler,
		},
		{
			MethodName: "StageTransactions",
			Handler:    _DarksideStreamer_StageTransactions_Handler,
		},
		{
			MethodName: "ApplyStaged",
			Handler:    _DarksideStreamer_ApplyStaged_Handler,
		},
		{
			MethodName: "ClearIncomingTransactions",
			Handler:    _DarksideStreamer_ClearIncomingTransactions_Handler,
		},
		{
			MethodName: "AddAddressUtxo",
			Handler:    _DarksideStreamer_AddAddressUtxo_Handler,
		},
		{
			MethodName: "ClearAddressUtxo",
			Handler:    _DarksideStreamer_ClearAddressUtxo_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StageBlocksStream",
			Handler:       _DarksideStreamer_StageBlocksStream_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "StageTransactionsStream",
			Handler:       _DarksideStreamer_StageTransactionsStream_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "GetIncomingTransactions",
			Handler:       _DarksideStreamer_GetIncomingTransactions_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "darkside.proto",
}
