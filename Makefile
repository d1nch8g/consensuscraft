
install:
	go mod tidy

# Build test binary for embedding
build-test-binary:
	go run main.go build-test-binary

# Run the FUSE encrypted filesystem demo
demo: build-test-binary
	go run main.go demo

# Clean up demo artifacts
clean:
	rm -f testbinary
	rm -rf encrypted_storage encrypted_fs

# Full demo run
run-demo: clean demo

.PHONY: install build-test-binary demo clean run-demo
