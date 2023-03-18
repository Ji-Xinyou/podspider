import sys

# use pod name as index
cpu = {}
mem = {}
net = {}
disk = {}

def parse_cpu_stat(s):
    podname = s[:s.find("|")]
    if podname not in cpu:
        cpu[podname] = {"cputime_nano": []}
    
    # record the history
    cputime_nano = int(s[s.find(" ") + 1:])
    cpu[podname]["cputime_nano"].append(cputime_nano)

def parse_mem_stat(s):
    podname = s[:s.find("|")]
    if podname not in mem:
        mem[podname] = {"rss": [], "cache": [], "swap": []}
    
    # record the history
    s = s[s.find("|") + 1:]
    rss = int(s[5:s.find("|")])

    s = s[s.find("|") + 1:]
    cache = int(s[7:s.find("|")])

    s = s[s.find("|") + 1:]
    swap = int(s[6:])

    mem[podname]["rss"].append(rss)
    mem[podname]["cache"].append(cache)
    mem[podname]["swap"].append(swap)

def parse_net_stat(s):
    podname = s[:s.find("|")]
    if podname not in net:
        net[podname] = {"rx_bytes": [], "tx_bytes": []}
    
    s = s[s.find("|") + 1:]
    rx_bytes = int(s[4:s.find("|")])

    s = s[s.find("|") + 1:]
    tx_bytes = int(s[5:])

    net[podname]["rx_bytes"].append(rx_bytes)
    net[podname]["tx_bytes"].append(tx_bytes)

def parse_disk_stat(s):
    podname = s[:s.find("|")]
    if podname not in disk:
        disk[podname] = {"read_bytes": [], "write_bytes": []}

    s = s[s.find("|") + 1:]
    read_bytes = int(s[10:s.find("|")])

    s = s[s.find("|") + 1:]
    write_bytes = int(s[10:])

    disk[podname]["read_bytes"].append(read_bytes)
    disk[podname]["write_bytes"].append(write_bytes)

def main():
    with open(sys.argv[1], "r") as f:
        lines = f.readlines()
        for line in lines:
            # remove the quote, and the newline
            log = line[line.find("msg=") + 5: -2]
            stat_header = log[:log.find(" ")]
            remaining = log[log.find("of") + 3:]
            if stat_header == "CPUStat":
                parse_cpu_stat(remaining)
            elif stat_header == "MemoryStat":
                parse_mem_stat(remaining)
            elif stat_header == "NetworkStat":
                parse_net_stat(remaining)
            elif stat_header == "DiskStat":
                parse_disk_stat(remaining)
            else:
                raise ValueError("Unknown stat header: " + stat_header)

if __name__ == "__main__":
    main()