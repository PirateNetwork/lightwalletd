protoc --go_out=. ./compact_formats.proto

rm compact_formats.pb.go

cp ./lightwalletd/walletrpc/compact_formats.pb.go ./compact_formats.pb.go

rm -rf ./lightwalletd
