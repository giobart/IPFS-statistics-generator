 # IPFS-statistics-generator

The application developed has 2 main purposes:

1.	Explore an IPFS node in order to extract some useful information about peers taxonomy in the swarm.  
2.	Exploit the DHT Query command in order to reconstruct the bucket-based recursion list of the query for a custom created CID.

Here a brief description of the application structure.

```raspberry-compile.sh``` <br>
This file contains a script that can be used in order to compile and deploy the application in a local raspberry pi. Is important to change user, hostname and path before the use. Is also important to place the DB files in the same folder with the binary in the remote machine. 

```IPV*-IP2LOCATION-*.BIN``` <br>
These files are the databases used for ip to region translation. They must be periodically updated and placed in the same folder with the executable binary.

```lib/charts.go``` <br>
Implementation of the handlers used to plot the graphs based on go-echarts library.  

```lib/cid-utils.go``` <br>
Utility functions used to manage the CID inside the program. The implementation of these functions is heavily based on the official IPFS go libraries. 

```lib/ipfs-interface.go``` <br>
Implementation of the HTTP calls to the IPFS node.

```lib/peer-geolocation.go``` <br>
Utility functions for peer geolocation. These functions are based on the ip2location-go library that uses the above DB files.

```lib/persistence.go``` <br>
Function used to store and retrieve data from the local database. The persistence has been realized with Scribble that is a JSON based DB.

# 1 - Discover Peer Taxonomy

The peers in the network changes frequently and with them also the composition itself of the swarm. As we’ll see there isn’t at all a fixed predictable number of peers in the swarm, instead, the swarm is very elastic and the nodes that compose the first line of contact of our node may vary a lot during the time.
The integration between the node and this application is implemented using the exposed HTTP Api. Even if exists a native go api, the choice of the HTTP approach was in a certain sense forced by the fact that the DHT-Query command, that we’ll use later on, is available only on the original CLI interface and through the HTTP interface.  

<b>When the application starts, inside the machine must be already up and running an IPFS node</b>, otherwise the application may crash. After the start-up process, the program start pulling data from the node, querying for the current swarm information. The pieces of information collected are then stored locally in JSON documents, ready to be used for the graph plotting operations.

In particular, the application stores the following data:

* For each Peer encountered:
    * CID
    *	Latency
    *	Address
    *	Nation
    *	City
    *	Latitude
    *	Longitude

* Every 5 minutes:
    * List of the peers composing the swarm
    * Timestamp

The application exposes a web page with the live charts, it’s possible to see the collected results after at least 5 minutes in the endpoint http://localhost:8081/stats

# 2 - DHT Query and Routing Tables 

The DHT Query command is part of the official CLI and HTTP API provided by the IPFS node. From the official documentation, given a PeerID, this command should return a list of the nearest peers to that id. In order to do so, this command start the query with the 3 nearest nodes that he knows (from his routing table) and ask them for a result. Each peer responds with a list of the peers that should be near the original CID. The definition of the word “near” resides in the definition of the Kademlia k-bucket. In fact, we can expect a response with a list of CIDs that resides in a k-bucket around the one where the CIDs belongs. <br>
The CID list received from the first 3 nodes is then added on the query and iteratively the node starts querying them. This command goes on until we just get timeout errors from the queried peers or non-relevant CIDs. <br>
We can imagine that of course under the hood this command uses iteratively the FindPeer primitive.<br><br>

The idea of this section of the program is to exploit this command in order to obtain a recursion graph of the network, bucket by bucket. To do so, the program has to generate ad-hoc CIDs that belongs to a specific k-bucket and start a new query with the generated CID. <br>
Since libp2p for the routing process generates the hash of the original peer id, in order to generate a CID that belongs to a specific bucket, the program must find a peer id that after the sha256 algorithm belongs to that specific bucket. <br>
This operation is very expensive in terms of computational power, and the machine used for this experiment wasn’t able to handle such a heavy computation. Therefore the implemented approach is slightly different. <br>
The algorithm performs a certain number of iterations, for the sake of the example I set it to 100k, and in each iteration generates a CID, create the hash, if it discovers a CID belonging to new bucket, it saves this CID for a future query. This algorithm has 2 interesting properties: bounded in time and effective. The former property is due to the fact that the number of iterations is limited and well known in advance. The latter property refers to the fact that since the Kademlia k-buckets emptiness increases for bucket with high id (the most distant ones) and since to find bucket with high id is exponentially increasingly hard, we concentrate the computational power to find, with high probability, buckets with low id, and just in case we are lucky, we find out bucket with high id. Of course, with only 100k iterations we must be very lucky, but with higher computational power we can also try several million iterations without problems. <br> 
<br>
Once we built a set of CIDs that we use to query the DHT, we can call the DHT query command (for convenience with a timeout of 10 seconds) in parallel on all the generated peer, and create the recursion graph for each bucket. In particular, the program examines all the responses of the DHT-Query command connecting each node with his own peer-list response recursively, in order to understand who is in the routing table of who.<br>
Unfortunately, reconstruct this graph at the beginning was very hard due to the lack of documentation provided for this command. In particular, the result to this query command is a list of JSON Object with the following fields: <br>

```
{
  "Extra": "<string>",
  "ID": "<peer-id>",
  "Responses": [
    {
      "Addrs": [
        "<multiaddr-string>"
      ],
      "ID": "peer-id"
    }
  ],
  "Type": "<int>"
}
```

* ID: represents the sender of the message in case of response message, the recipient otherwise
* Responses: represents the Peer list that the node has found as response to the given CID
* Type: represents the type of the message, the meaningful messages type for my purpose are:
    * Type 0: is a query message, this basically is the message that our peer sends to the recipient
    * Type 6: is a message that means that our peer is adding the provided ID to the iterative query system, this ID will be used as the next recipient for a query message
    * Type 1: is a response message from e Peer that received our query.
    

For each query, the program parses this list of responses in order to reconstruct the graph of the nodes and their respective connection for the specific bucket that the CID where constructed for.