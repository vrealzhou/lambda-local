#!/bin/bash

docker create -v $1:/var/lambdas -v $2:/var/lambdas/template.yaml --expose 3001:3001 --name lambda-local-go --network $3