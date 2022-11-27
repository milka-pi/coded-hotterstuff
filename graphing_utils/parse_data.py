from argparse import ArgumentParser
from time import sleep, time
import glob
import re

parser = ArgumentParser(description="Data processing")

parser.add_argument('--dir',
                     type=str,
                     help="Directory of the result files",
                     default=".")

parser.add_argument('--n',
                    type=int,
                    help="Number of nodes",
                    default="4")

#parser.add_argument('--m',
#                    type=str,
#                    help="mode: orig or coded broadcast",
#                    default="orig")

# parser.add_argument('--node',
#                     type=int,
#                     help="Index of the victim node to plot",
#                     default=0)

# Expt parameters
args = parser.parse_args()

#directory = '../experiments/' + args.m + '-' + str(args.n) + '-nodes/'

directory = args.dir+"/"
print(directory)
string_to_print = ""

traffic_re = re.compile(r"rxbytes=([0-9]+) txbytes=([0-9]+)$")
if __name__ == "__main__":
    files=glob.glob(directory + "*-traffic.log")
    print(files)
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
        string_to_print += "# time mbps\n"
        for i in range(N):
            print(i, datapoints[i])
            string_to_print += str(i) + " " + str(datapoints[i]) + "\n"
        print("\n")
        string_to_print += "\n\n"

f = open(directory + "data.dat", "w+")
f.write(string_to_print)
f.close()