#!/bin/bash
set -x #echo on

GOOS=linux go build -o divvy-trips main.go
