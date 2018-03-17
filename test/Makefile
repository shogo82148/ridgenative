help: ## Display this text
	# http://postd.cc/auto-documented-makefile/
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

setup: dist.zip ## Setup test environment in AWS
	./create-stack shogo82148-ridgenative template.yaml

update: dist.zip ## Update test environment in AWS
	./update-stack shogo82148-ridgenative template.yaml

test: dist.zip ## Do tests
	go test -v ./...

dist.zip: main.go
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -o dist/main main.go
	cd dist && zip ../dist.zip *

.PHONY: help setup update test

