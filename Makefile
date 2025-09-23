
# Install required dependencies
.PHONY: install
install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-bindata/go-bindata/...@latest
	npm install @minecraft/server

# Pack mod into single mcpack file locally
.PHONY: pack
pack:
	@echo "Creating mcpack file..."
	@rm -f x_ender_chest.mcpack
	@mkdir -p temp_pack
	@cp -r mod/behavior_pack temp_pack/
	@cp -r mod/resource_pack temp_pack/
	@cd temp_pack && zip -r ../x_ender_chest.mcpack . -x "*.DS_Store" "*/__pycache__/*"
	@rm -rf temp_pack
	@echo "Created x_ender_chest.mcpack"

# Unzip all zip files in current directory
.PHONY: unzip
unzip:
	@echo "Unzipping all zip files in current directory..."
	@for zipfile in *.zip; do \
		if [ -f "$$zipfile" ]; then \
			echo "Extracting $$zipfile..."; \
			unzip -o "$$zipfile"; \
		fi; \
	done
	@echo "All zip files extracted."

# Generate gRPC golang code
.PHONY: gen
gen: pack
	mkdir -p gen/pb
	protoc --go_out=. --go-grpc_out=. proto/consesnuscraft.proto
	go-bindata -o gen/xendchest/bindata.go -pkg xendchest x_ender_chest.mcpack
