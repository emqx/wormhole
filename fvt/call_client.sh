#!/bin/bash

go build -o fvt/edge_client main.go
chmod +x fvt/edge_client

export BUILD_ID=dontKillMe

pids=`ps aux|grep "edge_client" | grep "fvt"|awk '{printf $2 " "}'`
if [ "$pids" = "" ] ; then
   echo "No edge_client was started"
else
  for pid in $pids ; do
    echo "kill edge_client " $pid
    kill -9 $pid
  done
fi

nohup fvt/edge_client client $1 > edge_client.out 2>&1 &