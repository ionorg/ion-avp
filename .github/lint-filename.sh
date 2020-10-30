#!/usr/bin/env bash

#
# DO NOT EDIT THIS FILE
#
# It is automatically copied from https://github.com/pion/.goassets repository.
#
# If you want to update the shared CI config, send a PR to
# https://github.com/pion/.goassets instead of this repository.
#

set -e

SCRIPT_PATH=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
GO_REGEX="^[a-zA-Z][a-zA-Z0-9_]*\.go$"

find  "$SCRIPT_PATH/.." -name "*.go" | while read fullpath; do
  filename=$(basename -- "$fullpath")

  if ! [[ $filename =~ $GO_REGEX ]]; then
      echo "$filename is not a valid filename for Go code, only alpha, numbers and underscores are supported"
      exit 1
  fi
done
