.PHONY: install
install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

.PHONY: gen
gen:
	mkdir -p gen/pb
	protoc --go_out=. --go-grpc_out=. proto/consesnuscraft.proto

