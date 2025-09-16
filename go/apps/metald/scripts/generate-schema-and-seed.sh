#!/usr/bin/env bash

set -euo pipefail

BASE=$(dirname $(realpath $0))

cat "${BASE}/_schema.sql.preamble" | tee "${BASE}/../sqlc/schema.sql"

go run "${BASE}/netcalc.go" 10.0.0.0/8 /26 | tee -a "${BASE}/../sqlc/networks-seed.sql"
