#!/bin/bash
LAMBDA_ZIP_FILES=build/package/*.zip
TEMPLATE_FILE=deployments/aws/template.yaml
PORT=3001
ENV_JSON=env.json
NETWORK=network
CONTAINER_NAME=lambda-local-go

# Creates the container
docker pull vreal/lambda-local-go
docker create -v ~/.aws:/.aws --rm -p $PORT:3001/tcp --name $CONTAINER_NAME --network $NETWORK -e AWS_DEFAULT_PROFILE=default \
vreal/lambda-local-go:latest \
/var/lambdas/main -p HostEnv=$HOST_ENV -p DeployTime=2019-12-12 -p QueueStackName=QueueStack

# Copy template file to /var/lambdas/template.yaml
docker cp $TEMPLATE_FILE $CONTAINER_NAME:/var/lambdas/template.yaml

# Copy env json file to /var/lambdas/env.json
if [ -z "$ENV_JSON" ]
then
docker cp $ENV_JSON $CONTAINER_NAME:/var/lambdas/env.json
fi

# Copy lambda zip files to /var/lambdas
for f in $LAMBDA_ZIP_FILES
do
    echo "UPLOAD "$f
    docker cp $f $CONTAINER_NAME:/var/lambdas 
done

# Start lambda server
docker start --attach $CONTAINER_NAME