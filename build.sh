#!/bin/bash

env GOOS=linux GOARCH=amd64 go build -o ./bin/loadbalancer_linux main.go 
env GOOS=windows GOARCH=amd64 go build -o ./bin/loadbalancer_windows main.go
go build -o ./bin/loadbalancer_mac main.go

echo "Build complete."