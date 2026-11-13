#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
for experiment in companion frames; do
  (
    cd "$root/$experiment/harness"
    go test ./...
    if [ "$experiment" = companion ]; then
      go build -buildvcs=false -o ../probe ./cmd/probe
      cd ..
      python3 test_proof.py
    else
      go build -buildvcs=false -o ../frameprobe ./cmd/frameprobe
      cd ..
      python3 test_consumers.py
    fi
  )
done
