.PHONY: help build test race fmt vet run verify clean

BIN := bin/mini-datakit

help:
	@echo "make build   编译出 bin/mini-datakit"
	@echo "make test    跑全部单测"
	@echo "make race    带 -race 跑单测（W2 起必须）"
	@echo "make run     编译并运行采集端"
	@echo "make verify  提交前检查：fmt + vet + test + build"

build:
	go build -o $(BIN) ./cmd/agent

test:
	go test ./...

race:
	go test -race ./...

fmt:
	gofmt -l -w .

vet:
	go vet ./...

run: build
	./$(BIN) -config configs/agent.json

verify:
	bash scripts/verify.sh

clean:
	rm -rf bin
