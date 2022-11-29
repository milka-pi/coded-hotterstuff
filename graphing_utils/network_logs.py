import matplotlib.pyplot as plt
import os
import statistics
from pathlib import Path
from argparse import ArgumentParser

'''
Examples:
python3 network_logs.py --dir logs --savedir plots --n 7 --graph single --node 4
python3 network_logs.py --dir logs --savedir plots --n 4 --graph sum
python3 network_logs.py --dir logs --savedir plots --n 7 --graph all # fyi this looks absurd, I recommend reducing types to just 1 type if you want to do this
'''

ALL_NODES = "all"
SINGLE = "single"
SUM = "sum"
types = ["tcp_coded", "tcp_orig",  "udp_coded"]
# max = 100
max = 60
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

parser.add_argument('--node',
                    type=int,
                    help="node to graph",
                    default=1)

parser.add_argument('--graph',
                     type=str,
                     help="what to graph - single, sum, all",
                     default="single")

args = parser.parse_args()
n = args.n


# if any folder doesn't have .dat file, create it
for hotstuff_type in types:
    f = Path(f"{args.dir}/{hotstuff_type}_logs_{n}/data.dat")
    if not f.exists():
       os.system(f"python3 parse_data.py --n {n} --dir {args.dir}/{hotstuff_type}_logs_{n}")

data = {}

for hotstuff_type in types:
    f = open(f"{args.dir}/{hotstuff_type}_logs_{n}/data.dat", "r")
    data[hotstuff_type]={}
    node = -1
    for x in f:
        if x[0] == "#":
            node+=1
            data[hotstuff_type][node] = []
            continue
        if len(x.split()) < 2:
            continue
        data[hotstuff_type][node].append(float(x.split()[1]))

colors = ['c','m','y',"k"]
color_ind = 0
plt.figure(figsize=(10, 5))

def get_thpt_sum(vals):
    out = []
    for i in range(len(vals[0])):
        out.append(sum([vals[node][i] for node in range(n-1)]))
    return out

for hotstuff_type, vals in data.items():
    t = []
    for k, v in vals.items():
        t.extend(v)
    print(hotstuff_type, statistics.mean(t))
    if args.graph == ALL_NODES:
        for i in range(n):
            plt.plot(vals[i][:max], label=f'{hotstuff_type}_{i}')
    elif args.graph == SINGLE:
        plt.plot(vals[args.node][:max],colors[color_ind], label=hotstuff_type)
    elif args.graph == SUM:
        plt.plot(get_thpt_sum(vals)[:max], colors[color_ind], label=hotstuff_type)
    else:
        raise Exception("unexpected input to graph option")

    color_ind+=1

if args.graph == ALL_NODES:
    title = f'egress throughput for all nodes n={n}' 
    filename = f'thpt_all_nodes_{n}'
elif args.graph == SINGLE:
    title = f'egress throughput for node {args.node} n={n}'
    filename = f'thpt_node{args.node}_{n}' 
elif args.graph == SUM:
    title = f'sum of throughput for all nodes n={n}' 
    filename = f'thpt_sum_{n}'  
else:
    raise Exception("unexpected input to graph option")

plt.title(title)
plt.legend()
plt.savefig(f"plots/{filename}")
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
