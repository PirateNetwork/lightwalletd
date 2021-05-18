// Copyright (c) 2019-2020 The Zcash developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or https://www.opensource.org/licenses/mit-license.php .
package walletrpc

//go:generate protoc -I . --go_out=:../..  --go-grpc_out=:../.. ./compact_formats.proto
//go:generate protoc -I . --go_out=:../..  --go-grpc_out=:../.. ./service.proto
