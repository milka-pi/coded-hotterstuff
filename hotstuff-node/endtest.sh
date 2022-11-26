#!/usr/bin/bash

killall hotstuff-node
killall bmon
killall tcpcollect.sh
killall udpcollect.sh
bash ../../nanonet/script.sh stop
