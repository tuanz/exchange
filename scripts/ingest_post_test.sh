#!/bin/bash

#
# This script tests the outcomes of the ingest_test.sh,
# examining both JSON logs and the Pharos server to
# verify the following:
#
# 1. That items that should have been successfully ingested
#    were actually ingested.
# 2. That the various ingest processes set IngestManifest
#    attributes correctly in the JSON logs.
# 3. That for each ingested object, all files and events are
#    present in Pharos, and all attributes on all Pharos records
#    are correct.
# 4. That items that should have failed ingest did indeed fail,
#    and failed for the expected reasons.
#

cd ~/go/src/github.com/APTrust/exchange/integration

echo "Testing bucket reader output"
RUN_EXCHANGE_INTEGRATION=true go test -v apt_bucket_reader_test.go

echo "Testing apt_fetch output"
RUN_EXCHANGE_INTEGRATION=true go test -v apt_fetch_test.go
