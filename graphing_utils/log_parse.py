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

# parser.add_argument('--node',
#                     type=int,
#                     help="Which specific node id's log to parse",
#                     default=0)

args = parser.parse_args()
num_nodes = 4 
to_timestamp = lambda x: datetime.strptime(x,"%Y-%m-%dT%H:%M:%S.%fZ").timestamp() 

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
    # # first find first timestamp
    # for line in full_file:
    #     split_line = line.split()
    #     if len(split_line) < 2 or split_line[1] != LOG_LEVEL:
    #         continue
    #     log_start = to_timestamp(split_line[0])
    #     break
    # then iterate over whole file
    data = {}
    for n in range(num_nodes):
        process_file(num_nodes, n, data)
    pp = pprint.PrettyPrinter(indent=4)
    pp.pprint(data)
    print(len(data))
    per_block_data = {}
    for block_hash, node_info in data.items():
        print("============")
        decodes = []
        forwards = []
        peer_receieve = []
        leader_receieve = []
        peer_ready = []
        send = 0
        pp.pprint(node_info)
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
        per_block_data[block_hash]["last_decode"] = max(decodes) - per_block_data[block_hash]["send"] 
        per_block_data[block_hash]["last_ready"] = max(peer_ready) - per_block_data[block_hash]["send"] 
        # print(f"block hash: {block_hash}")
        # print(f"send time: {send}")
        # print(f"last receive from leader: {max(leader_receieve)-send}")
        # print(f"last receive from peer: {max(peer_receieve)-send}")
        # print(f"last decode: {max(decodes)-send}")
    pp.pprint(per_block_data)
