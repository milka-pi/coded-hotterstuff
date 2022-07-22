#!/usr/bin/bash


while getopts n: flag
do
    case "${flag}" in
        n) numNodes=${OPTARG};;
    esac
done

# Global variables
bw=10000


# 1st Step: Use nanonet script to create network nodes and extract their IP addresses

## create first network node (with idx=0)
msg=$(bash ../../nanonet/script.sh add 1 0 $bw)
msgarray=($msg)
ip=${msgarray[6]}
ipAddresses="$ip"

## create rest of network nodes (with idx >= 1)
for (( index=1; index<numNodes; index++ ))
do 
   msg=$(bash ../../nanonet/script.sh add 1 $index $bw)
   msgarray=($msg)
   ip=${msgarray[6]}
   ipAddresses+=",$ip"   
done

echo "List of IP addresses: $ipAddresses"


# 2nd Step: Run hotstuff-node for each node, passing the full list of IP addresses as an argument

for (( index=0; index<numNodes; index++ )); do
	ip netns exec ramjet-s1-n$index ./hotstuff-node -numNodes=$numNodes -index=$index -ipAddresses=$ipAddresses > hotstuff-$index.log &
	ip netns exec ramjet-s1-n$index bmon -o format:fmt='$(element:name) rxbytes=$(attr:rx:bytes) txbytes=$(attr:tx:bytes)\n' -p 'veth0' &> node-$index-traffic.log &
done















#msgarray=($msg)
#echo "Number of elements in msgarray: ${#msgarray[@]}"
#ip=${msgarray[6]}
#echo "IP address: $ip"

