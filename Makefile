GO=$(shell which go)
ENVIRONMENT=env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(FLAGS)
GOBUILD=$(ENVIRONMENT) $(GO) build -ldflags '-s -w'
GOCLEAN=$(GO) clean
PROFILE=default
DOCKER_VER=vreal/lambda-local-go:`date '+%Y%m%d%H%M'`

build-lambda:
	$(call build_lambda,Hello,hello)
	$(call build_lambda,Cheers,cheers)

build-docker:
	$(GOBUILD) -o build/docker/main cmd/docker/main.go
	docker rmi vreal/lambda-local-go || exit 0
	docker build -t vreal/lambda-local-go:latest build/docker/
	docker push vreal/lambda-local-go:latest
	docker tag vreal/lambda-local-go:latest $(DOCKER_VER)
	docker push $(DOCKER_VER)
	docker rmi $(DOCKER_VER)
	rm -rf build/docker/main

define build_lambda
	mkdir -p build/lambdas/$(2) && \
	$(GOBUILD) -o build/lambdas/$(2)/main test/lambdas/$(2)/*.go && \
	zip -j build/lambdas/$(1).zip build/lambdas/$(2)/main && \
	rm -rf build/lambdas/$(2)
endef