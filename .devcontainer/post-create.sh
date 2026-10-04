#!/usr/bin/env bash
# Runs once, when the dev container is created: the tools the Taskfile calls,
# pinned to the versions CI uses.
set -euo pipefail

# A workspace mounted from the host can belong to another user; git then
# refuses it as "dubious ownership", and every `go build` fails stamping VCS
# info ("error obtaining VCS status"). Trust this one directory.
git config --global --add safe.directory "$PWD"

go install github.com/go-task/task/v3/cmd/task@v3.49.1
# Built with the container's Go 1.27: golangci-lint refuses a config whose go
# directive is newer than the Go it was built with.
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
python3 -m pip install --user --no-warn-script-location -r requirements-docs.txt

echo
echo "Ready. Try: task rule -- R07"
