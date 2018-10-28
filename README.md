# lambda-local
This is a tool to run AWS Lambda locally and keep warm. The original perpose is for local testing AWS Lambda more effecient.

**NOTE:** 
* This program is only for test if Lambda function logic correct without deploy. It doesn't provide full feature of cloud version Lambdas.
* It has very limited feature which only match my work requirement. Anyone intested in it can make a fork and add yourself.

## Requitements
* go 1.11 or above
* docker v18 or above
* You need to setup valid credentials in ~/.aws/credentials file.

## Limits

* Only support Lambdas written in Go with [AWS Go Library](https://github.com/aws/aws-lambda-go)
* Lambdas should be defined in [AWS SAM yaml file](https://docs.aws.amazon.com/lambda/latest/dg/serverless_app.html)
* Same Lambda can only be invoked in a queue no matter the request side is concurrent or not.

## Install

Because this project is using go mod which shouldn't been cloned to GOPATH.

```shell
export GO111MODULE=on
go install github.com/vrealzhou/lambda-local
export GO111MODULE=auto
```

After installed you can use such command under your lambda project to start local test:
```shell
lambda-local start-lambda -t {SAM template file} -r {AWS region}
```

The first run of specified Lambda is a little bit slow but the second run will be fast.
