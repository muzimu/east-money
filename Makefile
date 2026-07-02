.PHONY: build clean test lint run

BINARY := eastmoney
CMD    := ./cmd/eastmoney

# 版本信息（可通过 go build -ldflags 注入）
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

# 默认目标
all: build

build:
	@echo "Building $(BINARY) $(VERSION) ($(GOOS)/$(GOARCH))..."
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

# 跨平台编译
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-amd64 $(CMD)

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-arm64 $(CMD)

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-darwin-amd64 $(CMD)

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-darwin-arm64 $(CMD)

build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-windows-amd64.exe $(CMD)

build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64
	@echo "All platforms built."

test:
	go test -count=1 ./...

test-short:
	go test -short -count=1 ./...

test-verbose:
	go test -v -count=1 ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY) $(BINARY)-*

run: build
	./$(BINARY) $(ARGS)
