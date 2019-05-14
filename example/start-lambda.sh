#!/bin/bash
LAMBDA_ZIP_FILES=build/package/*.zip
TEMPLATE_FILE=deployments/aws/template.yaml
PORT=3001
ENV_JSON=env.json
NETWORK=network

# Creates the container
docker rm lambda-local-go
cid=$(docker create -v ~/.aws:/.aws --rm -p $PORT:3001/tcp --name lambda-local-go --network $NETWORK -e AWS_DEFAULT_PROFILE=default \
vreal/lambda-local-go:latest \
/var/lambdas/main -p HostEnv=$HOST_ENV -p DeployTime=2019-12-12 -p QueueStackName=QueueStack)

# Copy template file to /var/lambdas/template.yaml
docker cp $TEMPLATE_FILE lambda-local-go:/var/lambdas/template.yaml

# Copy env json file to /var/lambdas/env.json
if [ -z "$ENV_JSON" ]
then
docker cp $ENV_JSON lambda-local-go:/var/lambdas/env.json
fi

# Copy lambda zip files to /var/lambdas
for f in $LAMBDA_ZIP_FILES
do
    echo "UPLOAD "$f
    docker cp $f lambda-local-go:/var/lambdas 
done

# Start lambda server
docker start $cid
# Attach to lambda server for outputs. For debug only
docker attach --no-stdin $cid