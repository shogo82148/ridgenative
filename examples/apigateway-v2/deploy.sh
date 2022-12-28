#!/bin/bash

set -eux

ROOT=$(cd "$(dirname "$0")" && pwd)
BUCKET=$1
STACK=$2

mkdir -p dist
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "$ROOT/dist/example" "$ROOT/main.go"

aws cloudformation package \
    --template-file "$ROOT/template.yaml" \
    --output-template-file "$ROOT/packaged.yaml" \
    --s3-bucket "$BUCKET"

aws cloudformation deploy \
    --stack-name "$STACK" \
    --template "$ROOT/packaged.yaml" \
    --capabilities CAPABILITY_IAM
