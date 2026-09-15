#!/bin/sh
# Builds the one file that goes to the Pi: the frontend compiles into static/,
# which main.go embeds, and the Go build cross-compiles for the Pi Zero W's
# ARMv6. CGO is off because modernc.org/sqlite is pure Go — no cross toolchain.
#
# Both binaries land in build/, which is gitignored, so nothing that gets built
# sits next to the source. build/api-armv6 is the file to copy to the Pi;
# build/sonic-finance-pi is the same code for this machine.
set -e

pnpm --dir web install --frozen-lockfile
pnpm --dir web build

mkdir -p build
go build -o build/sonic-finance-pi ./cmd
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -o build/api-armv6 ./cmd
