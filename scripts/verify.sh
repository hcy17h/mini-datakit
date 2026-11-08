#!/usr/bin/env bash
# 提交前的一键检查。任何一个环节失败就退出，不允许"带病提交"。
# 用法：bash scripts/verify.sh   （Windows 上请在 WSL / Git Bash 里跑）
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> 1/4 检查格式 (gofmt)"
unformatted="$(gofmt -l . || true)"
if [ -n "$unformatted" ]; then
  echo "以下文件没格式化，请先执行 make fmt："
  echo "$unformatted"
  exit 1
fi
echo "    ok"

echo "==> 2/4 静态检查 (go vet)"
go vet ./...
echo "    ok"

echo "==> 3/4 单元测试 (go test -race)"
go test -race ./...
echo "    ok"

echo "==> 4/4 编译"
go build -o bin/mini-datakit ./cmd/agent
echo "    ok"

echo ""
echo "全部通过 ✅"
