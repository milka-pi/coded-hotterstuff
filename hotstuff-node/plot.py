#!/usr/bin/python

import math
import os
import sys
import signal
from argparse import ArgumentParser
from time import sleep, time
import glob
import re
import json

parser = ArgumentParser(description="Data processing")
parser.add_argument('--dir',
                    type=str,
                    help="Directory of the result files",
                    default=".")

parser.add_argument('--node',
                    type=int,
                    help="Index of the victim node to plot",
                    default=0)

# Expt parameters
args = parser.parse_args()

traffic_re = re.compile(r"rxbytes=([0-9]+) txbytes=([0-9]+)$")
if __name__ == "__main__":
    files=glob.glob(args.dir+"/*-traffic.log")
    for filepath in files:
        datapoints = []
        lastRx = None
        with open(filepath) as f:
            for line in f:
                result = traffic_re.search(line)
                if not result is None:
                    rx = int(result.group(1))
                    tx = int(result.group(2))
                    if lastRx is None:
                        lastRx = tx
                    else:
                        datapoints.append((tx-lastRx)*8.0/1000000.0)    # mbps
                        lastRx = tx
        N = len(datapoints)
        print("# time", "mbps")
        for i in range(N):
            print(i, datapoints[i])
        print("\n\n")

