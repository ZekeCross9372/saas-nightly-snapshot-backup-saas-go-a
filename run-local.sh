#!/bin/sh
set -eu

go test ./...
go run . -once -input tenants.example.json
