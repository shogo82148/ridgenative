#!/bin/bash

set -eux

ROOT=$(cd "$(dirname "$0")" && pwd)

mkdir -p dist
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "$ROOT/dist/bootstrap" "$ROOT/main.go"

aws cloudformation package \
    --template-file "$ROOT/template.yaml" \
    --output-template-file "$ROOT/packaged.yaml" \
    --s3-bucket shogo82148-test \
    --s3-prefix ridgenative/examples/apigateway-v2

aws cloudformation deploy \
    --stack-name ridgenative-example-apigateway-v2 \
    --template "$ROOT/packaged.yaml" \
    --capabilities CAPABILITY_IAM
