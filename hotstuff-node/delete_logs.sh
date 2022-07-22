#!/usr/bin/bash

while getopts n: flag
do
    case "${flag}" in
        n) numNodes=${OPTARG};;
    esac
done

for (( idx=0; idx<numNodes; idx++ ))
do
	rm -f "./hotstuff-$idx.log"
	rm -f "./node-$idx-traffic.log"
done

