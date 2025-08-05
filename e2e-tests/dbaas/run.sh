#!bin/bash

REGION=$1
ENV=$2
API_KEY=$3
ENGINE_NAME=$4
ENGINE_VERSION=$5
STEP=$6
DB_PREFIX=$7
DB_PASSWORD=$8
DB_TYPE=$9

DIR_RUN="${REGION}/${DB_TYPE}/${ENGINE_NAME}_${ENGINE_VERSION}"
DIR_TF=$(dirname $BASH_SOURCE)

mkdir -p ${DIR_RUN}
find ${DIR_RUN} -name "*.tf" -delete

find "${DIR_TF}/common" -name "*.tf" -print0 | xargs -0 cp -t ${DIR_RUN}
find "${DIR_TF}/${ENGINE_NAME}/${DB_TYPE}" -name "step${STEP}.tf" -print0 | xargs -0 cp -t ${DIR_RUN}

cd ${DIR_RUN}
terraform apply -auto-approve -var="mgc_region=${REGION}" -var="env=${ENV}" -var="api_key=${API_KEY}" -var="engine_name=${ENGINE_NAME}" -var="engine_version=${ENGINE_VERSION}" -var="db_prefix=${DB_PREFIX}" -var="db_password=${DB_PASSWORD}"
cd -
