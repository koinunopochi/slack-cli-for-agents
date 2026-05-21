BIN := bin/slack

.PHONY: all build test lint fmt vet tidy clean install help

all: build

help:
	@echo "Targets:"
	@echo "  build   - go build -> $(BIN)"
	@echo "  test    - go test ./..."
	@echo "  lint    - go vet + gofmt check"
	@echo "  fmt     - gofmt -w ."
	@echo "  vet     - go vet ./..."
	@echo "  tidy    - go mod tidy"
	@echo "  clean   - rm bin/"
	@echo "  install - go install ./..."

build:
	mkdir -p bin
	go build -o $(BIN) .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

lint: vet
	@diff=$$(gofmt -l . | grep -v '^vendor/' || true); \
	if [ -n "$$diff" ]; then echo "gofmt needed:"; echo "$$diff"; exit 1; fi

tidy:
	go mod tidy

clean:
	rm -rf bin/

install:
	go install ./...
