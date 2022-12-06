from datetime import datetime
from argparse import ArgumentParser
import json
import pprint

LOG_LEVEL = "DEBUG"
# to_timestamp = lambda x: datetime.strptime(x,"%Y-%m-%dT%H:%M:%S.%fZ").timestamp() 

parser = ArgumentParser(description="log parsing")

parser.add_argument('--n',
                    type=int,
                    help="Which experiment to use (what # of nodes)",
                    default=4)

parser.add_argument('--dir',
                    type=str,
                    help="where to look for log folders",
                    default='logs_arch')

args = parser.parse_args()
num_nodes = 4 
to_timestamp = lambda x: datetime.strptime(x,"%Y-%m-%dT%H:%M:%S.%fZ").timestamp() 


# General order would be:

# Leader node sends the proposal - "send" log

# Follower has a recieve message from the leader (shareNumber = 0) - "recieve" log
# Follower forwards its part of the chunk - "forward" log
# Follower has enough sub-chunks to decode - "ready" log
# Follower decodes the sub-chunks into a proposal - "decode" log

def process_file(num_nodes, n, data):
    f = open(f"{args.dir}/tcp_coded_logs_microbenchmark_{num_nodes}/hotstuff-{n}.log", "r")
    full_file = f.read().split('\n')
    for line in full_file:
        whitespace_split_line = line.split()
        curly_split_line = line.split("{")
        if len(whitespace_split_line) < 2 or whitespace_split_line[1] != LOG_LEVEL or len(curly_split_line) <2:    
            continue
        json_str = "{"+curly_split_line[1]
        json_dict = json.loads(json_str)
        if "hash" in json_dict:
            block_hash = json_dict["hash"]
            timestamp = to_timestamp(whitespace_split_line[0])
            node = int(whitespace_split_line[2].split("=")[1])
            action = whitespace_split_line[4]
            data.setdefault(block_hash, {})
            data[block_hash].setdefault(node, {})
            if action in ["send", "decode", "ready"]:
                # have this if statement becuase I just keep the first time the action happens
                # since i don't get why it might happen twice haha
                if action not in data[block_hash][node]: 
                    data[block_hash][node][action] = timestamp
            elif action == "receive":
                share = json_dict["shareNumberField"]
                data[block_hash][node].setdefault(action, {})
                if share not in data[block_hash][node][action]: 
                    data[block_hash][node][action][share] = timestamp 
            elif action == "forward":
                to_node = json_dict["to_node"]
                data[block_hash][node].setdefault(action, {})
                if to_node not in data[block_hash][node][action]: 
                    data[block_hash][node][action][to_node] = timestamp 
        else:
            continue

if __name__ == "__main__":
    data = {}
    for n in range(num_nodes):
        process_file(num_nodes, n, data)
    
    pp = pprint.PrettyPrinter(indent=4)
    pp.pprint(data)
    print(len(data))
    per_block_data = {}

    #### AN ENTRY IN data DICT LOOKS LIKE THIS #####
    # KEY = block hash
    # dicts are actions that take place in the nodes & their time stamps
    # 's0+dCVQ5w8iE2Z+DCswdFLdP8QD8T++9aicJwYKZbok=': {   0: {   'decode': 1670386397.465, # time node 0 decoded this block
    #                                                        'forward': {   1: 1670386396.334, # time node 0 fwdd this block to node 1
    #                                                                       3: 1670386396.334},  # time node 0 fwdd this block to node 3
    #                                                        'ready': 1670386397.465,  # time node 0 was ready to decode this block
    #                                                        'receive': {   0: 1670386396.333, # time node 0 got its chunk part from leader
    #                                                                       1: 1670386397.464, # time node 0 got 1st chunk part from peer (not necessarily from node 1)
    #                                                                       2: 1670386399.063}}, # time node 0 got 2nd chunk part from peer 
    #                                                 1: {   'decode': 1670386397.25,
    #                                                        'forward': {   0: 1670386396.511,
    #                                                                       3: 1670386396.511},
    #                                                        'ready': 1670386397.249,
    #                                                        'receive': {   0: 1670386397.249,
    #                                                                       1: 1670386396.511,
    #                                                                       2: 1670386398.96}},
    #                                                 2: {   'send': 1670386395.154}, # time this block was sent (2 must have been the leader)
    #                                                 3: {   'decode': 1670386397.562,
    #                                                        'forward': {   0: 1670386397.733,
    #                                                                       1: 1670386397.733},
    #                                                        'ready': 1670386397.562,
    #                                                        'receive': {   0: 1670386397.325,
    #                                                                       1: 1670386397.562,
    #                                                                       2: 1670386397.733}}},

    for block_hash, node_info in data.items():
        # print("============")
        decodes = []
        forwards = []
        peer_receieve = []
        leader_receieve = []
        peer_ready = []
        send = 0
        per_block_data.setdefault(block_hash, {})
        for node, node_info in node_info.items():
            # print(node)
            for action, time in node_info.items():
                # print(action, time)
                if action == "send":
                    assert(send == 0)
                    send = time
                    per_block_data[block_hash]["send"] = time
                elif action == "receive":
                    if 0 in time:
                        leader_receieve.append(time[0])
                    for key, val in time.items():
                        if key!=0:
                            peer_receieve.append(val)
                elif action == "forward":
                    forwards.append(max(time.values()))
                elif action == "decode":
                    decodes.append(time)
                elif action == "ready":
                    peer_ready.append(time)
        if send == 0 or len(decodes) == 0:
            del per_block_data[block_hash]
            continue

        compare_func = max # min for first node to do it, max for last node to do it
        per_block_data[block_hash]["last_decode"] = compare_func(decodes) - per_block_data[block_hash]["send"] 
        per_block_data[block_hash]["last_ready"] = compare_func(peer_ready) - per_block_data[block_hash]["send"] 
        
        # You can uncomment this to visualize the data better perhaps
        # print(f"block hash: {block_hash}")
        # print(f"send time: {send}")
        # print(f"last receive from leader: {max(leader_receieve)-send}")
        # print(f"last receive from peer: {max(peer_receieve)-send}")
        # print(f"last decode: {max(decodes)-send}")
        
    rates = []
    for block_hash in per_block_data:
        last = per_block_data[block_hash]["last_ready"]
        rates.append(8/last)
