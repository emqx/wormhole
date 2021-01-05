#!/bin/bash

go build -o fvt/wormhole_agent main.go
chmod +x fvt/wormhole_agent

export BUILD_ID=dontKillMe

pids=`ps aux|grep "wormhole_agent" | grep "fvt"|awk '{printf $2 " "}'`
if [ "$pids" = "" ] ; then
   echo "No wormhole_agent was started"
else
  for pid in $pids ; do
    echo "kill wormhole_agent " $pid
    kill -9 $pid
  done
fi

nohup fvt/wormhole_agent client $1 > agent.out 2>&1 &