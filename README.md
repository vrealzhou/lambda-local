# lambda-local
This is a tool to run AWS Lambda locally and keep warm.

**NOTE:** This program has very limited feature which only match my work requirement. Anyone intested in it can make a fork and add yourself.

## Requitements
* go 1.11 or above
* docker v18 or above

## Install

Because this project is using go mod which shouldn't been cloned to GOPATH.

```shell
cd $NON_GOPATH
git clone github.com/vrealzhou/lambda-local.git
cd $NON_GOPATH/lambda-local
go install
```

After installed you can use such command under your lambda project to start local test:
```shell
lambda-local start-lambda -t {SAM template file} -r {AWS region}
```

The first run of specified Lambda is a little bit slow but the second run will be fast.