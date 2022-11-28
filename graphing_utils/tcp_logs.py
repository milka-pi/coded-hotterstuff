import matplotlib.pyplot as plt

TCP_CODED = "tcp_coded"
TCP_ORIGINAL = "tcp_orig"
PACING_RATE = "pacing_rate"
RTT = "rtt"
types = [TCP_CODED, TCP_ORIGINAL]
dir = "logs"
data_type = TCP_CODED
thing_to_graph = PACING_RATE

def process_pacing_rate(split):
    val = split[split.index(PACING_RATE)+1].split("bps")[0]
    if val[-1] == "M":
        return float(val[:-1])
    elif val[-1] == "k":
        return float(val[:-1])/1000 
    else:
        raise Exception(f"unexpected pacing rate {val}")

if __name__ == "__main__":
    colors = ["--bo","--ro","--go"]
    for i, n in enumerate([4,7,9]):
        f = open(f"{dir}/{data_type}_logs_{n}/node-1-tcp.log", "r")
        to_graph = []
        while True:
            line = f.readline()
            if not line:
                break
            split = line.strip().split()
            if len(split) == 0:
                continue
            if split[0] == "State":
                next = f.readline().split() # skip listen port info (2 lines)
                if next[0] != "LISTEN":
                    continue
                f.readline()
                this = []
                for _ in range(n-1):
                    f.readline() # skip ip
                    line = f.readline()
                    split = line.strip().split()
                    if len(split) == 0 or split[0] != "bbr":
                        raise Exception("line was not as expected")
                    if thing_to_graph == PACING_RATE:
                        if PACING_RATE not in split:
                            break
                        this.append(process_pacing_rate(split))
                    if thing_to_graph == RTT:
                        this.append(float(split[3].split(":")[1].split("/")[0]))
                if len(this) == (n-1):
                    to_graph.append(this)
        for j, vals in enumerate(to_graph):
            if len(vals) != (n-1):
                print(vals)
            plt.plot([j]*(n-1), vals, colors[i],label=n)
    plt.title(f'{thing_to_graph} {data_type}')
    handles, labels = plt.gca().get_legend_handles_labels()
    by_label = dict(zip(labels, handles))
    plt.legend(by_label.values(), by_label.keys())
    plt.show()
    # plt.savefig(f"plots/{thing_to_graph}_{data_type}_tcp")
            
        