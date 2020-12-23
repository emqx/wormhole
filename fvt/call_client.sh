#!/bin/bash

go build -o fvt/edge_client internal/client/client.go
chmod +x fvt/edge_client

export BUILD_ID=dontKillMe

nohup fvt/edge_client $1 > edge_client.out 2>&1 &