#!/usr/bin/bash

numberOfNodes=4
ipAddresses=""
echo "List of IP addresses: $ipAddresses"
for (( index=0; index<numberOfNodes; index++ ))
    do ./hotstuff-node -index=$index -ipAddresses=$ipAddresses & done
