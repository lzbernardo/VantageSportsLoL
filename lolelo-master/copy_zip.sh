#!/bin/bash

# This script updates the DataSpectator plugin in google cloud storage by
# first backing up the existing version in each environment and then copying
# the new one (see: make_zip.sh) to the expected location.

set -e

DATE=`date +%Y%m%d`

echo "Updating dev..."
gsutil mv gs://vsd-esports/lol/scripts/all/DataSpectator.zip gs://vsd-esports/lol/scripts/all/DataSpectator.zip.bak.$DATE
gsutil cp DataSpectator.zip gs://vsd-esports/lol/scripts/all/DataSpectator.zip

echo "Updating staging..."
gsutil mv gs://vss-esports/lol/scripts/all/DataSpectator.zip gs://vss-esports/lol/scripts/all/DataSpectator.zip.bak.$DATE
gsutil cp DataSpectator.zip gs://vss-esports/lol/scripts/all/DataSpectator.zip

echo "Updating prod..."
gsutil mv gs://vsp-esports/lol/scripts/all/DataSpectator.zip gs://vsp-esports/lol/scripts/all/DataSpectator.zip.bak.$DATE
gsutil cp DataSpectator.zip gs://vsp-esports/lol/scripts/all/DataSpectator.zip
