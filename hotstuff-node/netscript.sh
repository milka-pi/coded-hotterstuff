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

## delay the network
for (( i=0; i<numNodes; i++ ))
do
	for (( j=0; j<numNodes; j++ ))
	do
		if [[ $i -ne $j ]]; then
			bash ../../nanonet/script.sh delay 1 $i 1 $j 50
		fi
	done
done

echo "List of IP addresses: $ipAddresses"

# 2nd Step: Run hotstuff-node for each node, passing the full list of IP addresses as an argument

for (( index=0; index<numNodes; index++ )); do
	ip netns exec ramjet-s1-n$index ./hotstuff-node -numNodes=$numNodes -index=$index -ipAddresses=$ipAddresses &> hotstuff-$index.log &
	ip netns exec ramjet-s1-n$index bmon -o format:fmt='$(element:name) rxbytes=$(attr:rx:bytes) txbytes=$(attr:tx:bytes)\n' -p 'veth0' &> node-$index-traffic.log &
done
