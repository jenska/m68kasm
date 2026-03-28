BINARY := m68kasm
CMD_PATH := ./cmd/m68kasm

.PHONY: build test staticcheck lint release clean

build:
	go build $(CMD_PATH)

test:
	go vet ./...
	go test ./...

staticcheck:
	@command -v staticcheck >/dev/null || { \
		echo "staticcheck not installed. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
		exit 1; \
	}
	staticcheck ./...

lint: staticcheck

release: clean
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(BINARY)-linux-amd64 $(CMD_PATH)
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(BINARY)-linux-arm64 $(CMD_PATH)
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(BINARY)-darwin-amd64 $(CMD_PATH)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(BINARY)-darwin-arm64 $(CMD_PATH)
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(BINARY)-windows-amd64.exe $(CMD_PATH)
	GOOS=windows GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(BINARY)-windows-arm64.exe $(CMD_PATH)

clean:
	rm -rf dist
