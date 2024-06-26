#!/usr/bin/env bash

set -e

# set these 2 environment variables
if [ -z $CAUSE_DT_BUG_ENDPOINT ]; then
  echo "Set DT endpoint URL as env var CAUSE_DT_BUG_ENDPOINT"
  exit 1
fi

if [ -z $CAUSE_DT_BUG_API_KEY ]; then
  echo "Set DT API key as env var CAUSE_DT_BUG_API_KEY"
  exit 1
fi

PARENT_PROJECT_ID=`curl --request PUT \
  --url "$CAUSE_DT_BUG_ENDPOINT/api/v1/project" \
  --header "X-API-Key: $CAUSE_DT_BUG_API_KEY" \
  --header "Content-Type: application/json" \
  --data "{\"name\":\"test-parent-project\",\"classifier\":\"APPLICATION\",\"active\":true}"  | jq -r '.uuid'`

echo "PARENT_PROJECT_ID=$PARENT_PROJECT_ID"

CHILD_PROJECT_ID=`curl --request PUT \
  --url "$CAUSE_DT_BUG_ENDPOINT/api/v1/project" \
  --header "X-API-Key: $CAUSE_DT_BUG_API_KEY" \
  --header "Content-Type: application/json" \
  --data "{\"name\":\"test-child-project\",\"classifier\":\"APPLICATION\",\"active\":true,\"parent\":{\"uuid\":\"$PARENT_PROJECT_ID\"}}"  | jq -r '.uuid'`

echo "CHILD_PROJECT_ID=$CHILD_PROJECT_ID"

echo "GET parent response:"

curl --request GET \
  --url "$CAUSE_DT_BUG_ENDPOINT/api/v1/project/$PARENT_PROJECT_ID" \
  --header "X-API-Key: $CAUSE_DT_BUG_API_KEY" \

echo

echo "GET child response:"

curl --request GET \
  --url "$CAUSE_DT_BUG_ENDPOINT/api/v1/project/$CHILD_PROJECT_ID" \
  --header "X-API-Key: $CAUSE_DT_BUG_API_KEY" \

echo

# can be uncommented to avoid duplicate names in repeated testing
#curl --request DELETE \
#  --url "$CAUSE_DT_BUG_ENDPOINT/api/v1/project/$PARENT_PROJECT_ID" \
#  --header "X-API-Key: $CAUSE_DT_BUG_API_KEY" \
