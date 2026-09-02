#!/bin/sh
# Builds the one file that goes to the Pi: the frontend compiles into static/,
# which main.go embeds, and the Go build cross-compiles for the Pi Zero W's
# ARMv6. CGO is off because modernc.org/sqlite is pure Go — no cross toolchain.
set -e

npm --prefix web ci
npm --prefix web run build

CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -o api-armv6 .
