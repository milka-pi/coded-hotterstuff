#!/bin/sh

numberOfNodes=4
for (( index=0; index<numberOfNodes; index++ ))
    do ./hotstuff-node -index=$index & done
