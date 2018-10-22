GO=$(shell which go)
ENVIRONMENT=env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(FLAGS)
GOBUILD=$(ENVIRONMENT) $(GO) build -ldflags '-s -w'
GOCLEAN=$(GO) clean
PROFILE=default

build-lambda:
	mkdir -p build/lambdas/Test
	$(GOBUILD) -o build/lambdas/Test/main test/lambda/main.go
	zip -j build/lambdas/Test.zip build/lambdas/Test/main
	rm -rf build/lambdas/Test

build-docker:
	$(GOBUILD) -o build/docker/main cmd/docker/main.go
	docker rmi lambda-local-go || exit 0
	docker build -t lambda-local-go build/docker/
	rm -rf build/docker/main
