#!/bin/bash
# generate server, client and worker code using the protocol buffer compiler
modules=( client server worker )
for item in "${modules[@]}"
do
    echo "Generating $item stub..."
    cd $item
    protoc --go_out=. \
           --go_opt=paths=source_relative \
           --go-grpc_out=. \
           --go-grpc_opt=paths=source_relative \
           intf/proto$item.proto
    echo "Done."
    echo ""
    cd ..
done

# arrange dependencies
echo "Arranging module with relative dependencies..."
go mod init distgrep
go mod tidy
echo "Done."
echo ""    
    
# compile entities
for item in "${modules[@]}"
do
    cd $item
    echo "Building $item..."
    go build -v -o bin/$item
    echo "Done."
    echo ""
    cd ..
done

