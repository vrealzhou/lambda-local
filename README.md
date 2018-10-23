# lambda-local
This is a tool to run AWS Lambda locally and keep warm.

**NOTE:** This program has very limited feature which only match my work requirement. Anyone intested in it can make a fork and add yourself.

## Requitements
* go 1.11 or above
* docker v18 or above

## Install

```shell
go get github.com/vrealzhou/lambda-local
cd $GOPATH/src/github.com/vrealzhou/lambda-local
go install
```

After installed you can use such command under your lambda project to start local test:
```shell
lambda-local start-lambda -t {SAM template file} -r {AWS region}
```

The first run of specified Lambda is a little bit slow but the second run will be fast.