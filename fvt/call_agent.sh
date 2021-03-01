#!/bin/bash

nohup fvt/wormhole_agent client $1 > agent_client.out 2>&1 &
