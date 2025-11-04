#!/bin/bash

# デプロイスクリプト - m3u-sync
# ビルドしたファイルをリモートサーバーに配置

set -e

# デフォルト設定
DEFAULT_TARGET="shinderuman@shinderumanm.local:~/.local/bin/"
DEFAULT_BINARY="m3u-sync"

# 引数の設定（デフォルト値を使用）
TARGET=${1:-$DEFAULT_TARGET}
BINARY=${2:-$DEFAULT_BINARY}

echo "Building $BINARY..."
go build -o $BINARY .

echo "Setting executable permissions..."
chmod +x $BINARY

echo "Deploying $BINARY to $TARGET..."
scp $BINARY $TARGET

echo "Cleaning up local binary..."
rm $BINARY

echo "Deployment completed successfully!"
echo "Binary deployed to: $TARGET"