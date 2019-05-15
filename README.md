# lambda-local
This is a tool to run AWS Lambda locally and keep warm. The original perpose is for local testing AWS Lambda more effecient.

**NOTE:** 
* This program is only for test if Lambda function logic correct without deploy. It doesn't provide full feature of cloud version Lambdas.
* It has very limited feature which only match my work requirement. Anyone intested in it can make a fork and add yourself.

## Requitements
* go 1.12 or above
* docker v2.0.0.0 or above
* You need to setup valid credentials in ~/.aws/credentials file.

## Limits

* Only support Lambdas written in Go with [AWS Go Library](https://github.com/aws/aws-lambda-go)
* Lambdas should be defined in [AWS SAM yaml file](https://docs.aws.amazon.com/lambda/latest/dg/serverless_app.html)
* Same Lambda can only be invoked in a queue no matter the request side is concurrent or not.

## Install

It's a web service running in docker container. You can run a shell script to setup/start the container. Full example file is under example/start-lambda.sh. You need to cover those steps to start the container correctly.

1. Download docker image 
    ```
    docker pull vreal/lambda-local-go
    ```
2. Create a container from this image. The command is just an example, You can create your own version. The `/var/lambdas/main` can take multiple `-p` options to override the yaml template parameters. `HostEnv` and `CacheHost` are just example which may not necessary in your own projects. `-e AWS_DEFAULT_PROFILE=default` is necessary if you are not using default AWS credentials profile.
    ```
    docker create -v ~/.aws:/.aws --rm -p $PORT:3001/tcp --name $CONTAINER_NAME --network $NETWORK \
    -e AWS_DEFAULT_PROFILE=default -e OTHER_ENV_VARIABLE=$other_variable \
    vreal/lambda-local-go:latest \
    /var/lambdas/main -p HostEnv=$HOST_ENV -p CacheHost=$CACHE_HOST
    ```

3. Copy template file to /var/lambdas/template.yaml:
    ```
    docker cp $TEMPLATE_FILE $CONTAINER_NAME:/var/lambdas/template.yaml
    ```

4. Copy env json file to /var/lambdas/env.json if exists:
    ```
    if [ -z "$ENV_JSON" ]
    then
    docker cp $ENV_JSON $CONTAINER_NAME:/var/lambdas/env.json
    fi
    ```

5. Copy lambda zip files to /var/lambdas
    ```
    for f in $LAMBDA_ZIP_FILES
    do
        echo "UPLOAD "$f
        docker cp $f $CONTAINER_NAME:/var/lambdas 
    done
    ```

6. Start container. The example turned on the attach to print the logs for debugging. You can turn it off if you don't need to see the logs.
    ```
    docker start --attach $CONTAINER_NAME
    ```

The first request of specified Lambda is a little bit slow but the second time will be faster.

