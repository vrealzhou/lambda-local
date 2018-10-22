GO=$(shell which go)
ENVIRONMENT=env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(FLAGS)
GOBUILD=$(ENVIRONMENT) $(GO) build -ldflags '-s -w'
GOCLEAN=$(GO) clean
PROFILE=default

build-lambda:
	mkdir -p build/lambdas/Test
	$(GO) build -ldflags '-s -w' -o build/lambdas/Test/main test/lambda/main.go
	zip -j build/lambdas/Test.zip build/lambdas/Test/main
	rm -rf build/lambdas/Test