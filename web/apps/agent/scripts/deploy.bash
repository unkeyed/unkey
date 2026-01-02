#!/bin/bash


regionsResponse=$(fly platform regions --json)

count=$(echo $regionsResponse | jq '. | length')

# returns a comma delimited list of regions for fly cli: 'iad,ord,dfw,...'
commaDelimitedRegions=$(echo $regionsResponse | jq  '.[].Code' | paste -sd "," - | sed 's/"//g')

 fly --config=fly.production.toml scale count $count --max-per-region=1 --region=$commaDelimitedRegions