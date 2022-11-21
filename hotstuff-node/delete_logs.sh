#!/usr/bin/bash

while getopts n: flag
do
    case "${flag}" in
        n) numNodes=${OPTARG};;
    esac
done

for (( idx=0; idx<numNodes; idx++ ))
do
	rm -f "./logs/hotstuff-$idx.log"
	rm -f "./logs/node-$idx-traffic.log"
        rm -f "./logs/node-$idx-tcp.log"
done

