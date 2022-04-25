#!/bin/sh

numberOfNodes=4
ipAddresses=""
for (( index=0; index<numberOfNodes; index++ ))
    do ./hotstuff-node -index=$index -ip=$ipAddresses & done
