#!/bin/bash

REGION=$1
ENV=$2
API_KEY=$3
JOB_ID=$4

export MGC_API_KEY=$API_KEY
mgc config set env ${ENV}
mgc config set region ${REGION}

STATUS_NOT_DELETABLE=('"DELETING"' '"BACKING_UP"')

###############################################################################
# DBaaS Instances Cleanup
###############################################################################
for instance in $(mgc dbaas instances list --raw --output json --api-key ${API_KEY} | jq -c .results[]); do
  id=$(echo ${instance} | jq .id)
  name=$(echo ${instance} | jq .name)
  status=$(echo ${instance} | jq .status)
  created_at=$(echo ${instance} | jq .created_at --raw-output | grep -Po '^\d{4}-\d{2}-\d{2}')

  IS_JOB_ID=$(echo $name | awk -v job_id="${JOB_ID}" '{ regex_pattern="^\"ci-github-"job_id".+-terraform-.*\"$"; if (match($0, regex_pattern)) { print "1" } else { print "0" } }')
  if [[ $IS_JOB_ID = 1 ]]; then
    echo "Instance id ${id} name ${name} with status ${status} created at ${created_at}"

    if [[ $(printf "%s\n" "${STATUS_NOT_DELETABLE[@]}" | grep -o "${status}" | wc -w) = 1 ]]; then
      echo "Can not be deleted because of ${status}"
    else
      echo "Deleting..."
      mgc dbaas instances delete --instance-id ${id} --no-confirm --raw --output json --api-key ${API_KEY}
    fi
  fi
done

###############################################################################
# DBaaS Replicas Cleanup
###############################################################################
for replica in $(mgc dbaas replicas list --raw --output json --api-key ${API_KEY} | jq -c .results[]); do
  id=$(echo ${replica} | jq .id)
  name=$(echo ${replica} | jq .name)
  status=$(echo ${replica} | jq .status)
  created_at=$(echo ${replica} | jq .created_at --raw-output | grep -Po '^\d{4}-\d{2}-\d{2}')

  IS_JOB_ID=$(echo $name | awk -v job_id="${JOB_ID}" '{ regex_pattern="^\"ci-github-"job_id".+-terraform-.*\"$"; if (match($0, regex_pattern)) { print "1" } else { print "0" } }')
  if [[ $IS_JOB_ID = 1 ]]; then
    echo "Replica id ${id} name ${name} with status ${status} created at ${created_at}"

    if [[ $(printf "%s\n" "${STATUS_NOT_DELETABLE[@]}" | grep -o "${status}" | wc -w) = 1 ]]; then
      echo "Can not be deleted because of ${status}"
    else
      echo "Deleting..."
      mgc dbaas replicas delete --replica-id ${id} --no-confirm --raw --output json --api-key ${API_KEY}
    fi
  fi
done

###############################################################################
# DBaaS Clusters Cleanup
###############################################################################
for cluster in $(mgc dbaas clusters list --raw --output json --api-key ${API_KEY} | jq -c .results[]); do
  id=$(echo ${cluster} | jq .id)
  name=$(echo ${cluster} | jq .name)
  status=$(echo ${cluster} | jq .status)
  created_at=$(echo ${cluster} | jq .created_at --raw-output | grep -Po '^\d{4}-\d{2}-\d{2}')

  IS_JOB_ID=$(echo $name | awk -v job_id="${JOB_ID}" '{ regex_pattern="^\"ci-github-"job_id".+-terraform-.*\"$"; if (match($0, regex_pattern)) { print "1" } else { print "0" } }')
  if [[ $IS_JOB_ID = 1 ]]; then
     echo "Cluster id ${id} name ${name} with status ${status} created at ${created_at}"

    if [[ $(printf "%s\n" "${STATUS_NOT_DELETABLE[@]}" | grep -o "${status}" | wc -w) = 1 ]]; then
      echo "Can not be deleted because of ${status}"
    else
      echo "Deleting..."
      mgc dbaas clusters delete --cluster-id ${id} --no-confirm --raw --output json --api-key ${API_KEY}
    fi
  fi
done
