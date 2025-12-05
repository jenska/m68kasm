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
	GOOS=linux GOARCH=amd64 go build -o dist/$(BINARY) $(CMD_PATH)

clean:
	rm -rf dist
