from datetime import datetime
from argparse import ArgumentParser

LOG_LEVEL = "INFO"
to_timestamp = lambda x: datetime.strptime(x,"%Y-%m-%dT%H:%M:%S.%fZ").timestamp() 

parser = ArgumentParser(description="log parsing")

parser.add_argument('--n',
                    type=int,
                    help="Which experiment to use (what # of nodes)")

parser.add_argument('--dir',
                    type=str,
                    help="where to look for log folders",
                    default='logs')

parser.add_argument('--node',
                    type=int,
                    help="Which specific node id's log to parse",
                    default=0)

args = parser.parse_args()
n = args.n
to_timestamp = lambda x: datetime.strptime(x,"%Y-%m-%dT%H:%M:%S.%fZ").timestamp() 

if __name__ == "__main__":
    f = open(f"{args.dir}/tcp_coded_logs_{n}/hotstuff-{args.node}.log", "r")
    full_file = f.read().split('\n')
    # first find first timestamp
    for line in full_file:
        split_line = line.split()
        if len(split_line) < 2 or split_line[1] != LOG_LEVEL:
            continue
        log_start = to_timestamp(split_line[0])
        break
    for line in full_file:
        split_line = line.split()
        if len(split_line) < 2 or split_line[1] != LOG_LEVEL:    
            continue
        timestamp = to_timestamp(split_line[0])
        time_since_start = timestamp-log_start

