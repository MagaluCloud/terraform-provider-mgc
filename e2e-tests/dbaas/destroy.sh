#!bin/bash

set -euo pipefail

REGION=$1
ENV=$2
API_KEY=$3
ENGINE_NAME=$4
ENGINE_VERSION=$5
DB_TYPE=$6

DIR_RUN="${REGION}/${DB_TYPE}/${ENGINE_NAME}_${ENGINE_VERSION}"

cd ${DIR_RUN}
terraform destroy -auto-approve -var="mgc_region=${REGION}" -var="env=${ENV}" -var="api_key=${API_KEY}"
cd -
