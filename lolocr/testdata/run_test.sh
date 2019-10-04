#!/bin/bash
set -e

pushd $GOPATH/src/github.com/VantageSports/ocr/lolocr/testdata

# Test 1: Run real ocr on a short "death" clip.
time python ../generate_states.py -i ./2190082947-na-p7-deathclip.mp4 -p 7 -f 20 -o ./2190082947-na-p7-deathclip-ocr.json --pretty
time python ../post_process.py -i ./2190082947-na-p7-deathclip-ocr.json -s correct -p 7 -o ./2190082947-na-p7-deathclip-corrected.json --pretty

# Test 2: run the match_align test with the first 5 minutes of a pre-computed game
time python ../post_process.py -i ./2190082947-na-p7-first9min-ocr.json -m ./2190082947-na-riot.json -s baseview -p 7 -u 27049826 -o ./2190082947-na-p7-first9min-events.json --pretty

# Test 3: run the benchmark script and save the results.
#type python ./benchmark ...

popd
