#!/bin/bash
set -e

touch DataSpectator.zip && rm ./DataSpectator.zip
rm -rf DataSpectator/bin/* DataSpectator/obj/*;
zip -r -X ./DataSpectator.zip DataSpectator/*