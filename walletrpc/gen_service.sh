protoc --go_out=. ./service.proto
protoc --go-grpc_out=. ./service.proto

rm service.pb.go
rm service_grpc.pb.go

cp ./lightwalletd/walletrpc/service.pb.go ./service.pb.go
cp ./lightwalletd/walletrpc/service_grpc.pb.go ./service_grpc.pb.go

rm -rf ./lightwalletd
