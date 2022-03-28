#!/bin/sh

numberOfNodes=3
for (( index=0; index<numberOfNodes; index++ ))
    do ./node -index=$index -sendMsg=true & done
