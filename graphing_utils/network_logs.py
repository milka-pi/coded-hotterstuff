import matplotlib.pyplot as plt
import os
from pathlib import Path
from argparse import ArgumentParser

types = ["tcp_coded", "tcp_orig",  "udp_coded", "kev_coded"]

parser = ArgumentParser(description="Data processing")

parser.add_argument('--savedir',
                     type=str,
                     help="Directory to save graphs",
                     default=".")

parser.add_argument('--dir',
                     type=str,
                     help="Directory of the result files",
                     default=".")

parser.add_argument('--n',
                    type=int,
                    help="Number of nodes")

args = parser.parse_args()

n = args.n


# if any folder doesn't have .dat file, create it
for log_type in types:
    f = Path(f"{args.dir}/{log_type}_logs_{n}/data.dat")
    if not f.exists():
       os.system(f"python3 parse_data.py --n {n} --dir {args.dir}/{log_type}_logs_{n}")

data = {}

for log_type in types:
    f = open(f"{args.dir}/{log_type}_logs_{n}/data.dat", "r")
    data[log_type]={}
    node = 0
    for x in f:
        if x[0] == "#":
            node+=1
            data[log_type][node] = []
            continue
        if len(x.split()) < 2:
            continue
        data[log_type][node].append(float(x.split()[1]))
    print(data)

colors = ['r','c','b',"k"]
color_ind = 0
plt.figure(figsize=(10, 5))
for type, vals in data.items():
    plt.plot(vals[2][:100],colors[color_ind], label=type)
    color_ind+=1
plt.title(f'throughput n = {n}')
plt.legend()
plt.savefig(f"plots/{n}")
# plt.show()

# You can use this to compare 
# for type, vals in data.items():
    # color_ind = 0
    # for i in range(1,n+1):
    #     plt.plot(vals[i],colors[color_ind],label=i)
    #     color_ind+=1
    # plt.title(type)
    # plt.legend()
    # plt.show()
