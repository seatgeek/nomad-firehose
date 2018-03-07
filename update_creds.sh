#!/bin/bash

echo "Usage: update_creds.sh env profile_name"
echo "For example: ./update_creds.sh dev Deveveloper"

set -u

echo "AWS_ACCOUNT_ID=$(aws sts get-caller-identity --output text --query 'Account' --profile $2)" > build-$1.env
echo "AWS_REGION=$(aws configure get region --profile $2)" >> build-$1.env 
echo "AWS_DEFAULT_REGION=$(aws configure get region --profile $2)" >> build-$1.env
echo "AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id --profile $2)" >> build-$1.env
echo "AWS_SECRET_ACCESS_KEY=$(aws configure get aws_secret_access_key --profile $2)" >> build-$1.env

AWS_SESSION_TOKEN="$(aws configure get aws_session_token --profile $2)"
# Save token only if it exists (when STS is used for temp creds)
echo "$AWS_SESSION_TOKEN" | grep . && echo "AWS_SESSION_TOKEN=$AWS_SESSION_TOKEN" >> build-$1.env

jet encrypt build-$1.env build-$1.env.encrypted

echo "Updated and encrypting creds for env $1, in build-$1.env.encrypted"
