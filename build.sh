#!/bin/sh

PACKAGES="."
COMMANDS="cmd/*.go"
LINTABLE="./logparse ./cmd/sshsessions.go"

set -e

go test $PACKAGES
go vet $PACKAGES
go fmt $PACKAGES
#golint $PACKAGES

for COMMAND in $COMMANDS; do
  go build $COMMAND
  go vet $COMMAND
  go fmt $COMMAND
  #golint $COMMAND
done

echo "SUCCESS"
