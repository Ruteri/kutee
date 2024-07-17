## Oram Store

1. Make sure you have libodsl installed (see libodsl section)
2. To run the oram store service, simply run 
	`LD_LIBRARY_PATH=/usr/local/lib go run ./cmd/httpserver/main.go`
3. To check the store with a client run:
	`go run ./cmd/cli/main.go set --key <key> --value <value>`
	`go run ./cmd/cli/main.go get --key <key>`
	And you should see the value returned!

## libodsl

This example shows how to compile the C++ codes into a shared library, and called by a Go program. Wrapper codes are generated using the Swig tool.

1. Make sure Go, Swig, and bearssl are installed. Change go.mod with the correct Go version.
2. Switch to the odsl folder.
3. Run $ `make all` to build and install libodsl
4. Run $ `LD_LIBRARY_PATH=/usr/local/lib go test ./...` to see it work!
