#!/bin/bash

go clean; go build

./websocket-proxy -jwt-public-key-file="$CATTLE_HOME/api.crt" -listen-address="localhost:8080" -cattle-address="localhost:8081"
