package lib

type RoutingTable struct {
	Buckets [256]Bucket
}

type Bucket struct {
	PeersId  [20]string
	BucketId int
}

/*query the dht in order to find out the routing table of the node */
func RoutingTableExplore() RoutingTable {

	var results = make(chan Bucket)
	var routingTable = RoutingTable{}

	//for each bucket id, generate estimation of the content
	for i := 0; i < 256; i++ {
		go extractBucket(i, results)
	}
	//collect all the 256 query resulting buckets and assign them to the routing table
	for i := 0; i < 256; i++ {
		bucket := <-results
		routingTable.Buckets[bucket.BucketId] = bucket
	}

	return routingTable
}

/* estimate the bucket i of the routing table exploiting the calls effectuated by the dht-query ipfs command  */
func extractBucket(bucketId int, responseChan chan<- Bucket) {
	queryCid, err := BucketPrefixBuilder(GetMyCid(), bucketId)
	if err != nil {
		log.Error("Invalid cid generated by PrefixBuilder: ", err)
	}
	print(queryCid)
	//TODO: run dht-query
	//TODO: extract first type 6 calls, they are the one that our node performed with the infos in is routing table
	//TODO: build the bucket structure
}