#!/usr/bin/bash

for i in {10..12}
do
   bash netscript.sh -n $i
   sleep 2
done
