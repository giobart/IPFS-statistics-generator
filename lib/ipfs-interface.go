package lib

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// Url of the endpoint exposed for the ipfs swarm list api
// used to extract the list of all the node in the current swarm
const ipfsSwarmListHttpUrl = "http://127.0.0.1:5001/api/v0/swarm/peers?verbose=true"
const ipfsGetCidHttpUrl = "http://localhost:5001/api/v0/id"
const ipfsDhtQueryHttpUrl = "http://localhost:5001/api/v0/dht/query"

/* using ipfs HTTP api -> gets the list of the current connected peer to the swarm */
func SwarmStatusList() []Peer {
	var result = make([]Peer, 1)

	//http request to ipfs api
	resp, err := http.Get(ipfsSwarmListHttpUrl)
	if err != nil {
		//not connection available, retry later
		log.Error("No available connection to ipfs:")
		log.Error(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//error reading body content of the request
		log.Error("Unable to read body content of swarm status list: ")
		log.Error(err)
	}

	var peer_list map[string]interface{}

	jsonErr := json.Unmarshal(body, &peer_list)
	if jsonErr != nil {
		//error parsing the json response
		log.Error(jsonErr)
	}

	for _, peer := range peer_list["Peers"].([]interface{}) {
		currPeer := peer.(map[string]interface{})
		p := Peer{}
		p.Addr = currPeer["Addr"].(string)
		p.Cid = currPeer["Peer"].(string)
		p.Latency = currPeer["Latency"].(string)
		result = append(result, p)
	}

	return result
}

func GetMyCid() string {
	//http request to ipfs api
	resp, err := http.Get(ipfsGetCidHttpUrl)
	if err != nil {
		//not connection available, retry later
		log.Error("No available connection to ipfs:")
		log.Error(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//error reading body content of the request
		log.Error("Unable to read body content of swarm status list: ")
		log.Error(err)
	}

	var peerInfo map[string]interface{}

	jsonErr := json.Unmarshal(body, &peerInfo)
	if jsonErr != nil {
		//error parsing the json response
		log.Error(jsonErr)
	}

	return peerInfo["ID"].(string)
}

func DhtQuery(cid string) []map[string]interface{} {
	//http request to ipfs api
	url := ipfsDhtQueryHttpUrl + "?arg=" + cid
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(url)

	if err != nil {
		//not connection available, retry later
		log.Error("No available connection to ipfs:")
		log.Error(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//error reading body content of the request
		log.Error("Unable to read body content of swarm status list: ")
		log.Error(err)
	}

	recursion_list := make([]map[string]interface{}, 0)

	for _, row := range strings.Split(string(body), "\n") {
		var elem map[string]interface{}
		jsonErr := json.Unmarshal([]byte(row), &elem)
		if jsonErr != nil {
			//error parsing the json response
			log.Error(jsonErr)
		}
		recursion_list = append(recursion_list, elem)
	}

	return recursion_list
}

// using dht query ipfs command, try to explore the recursion
func GetDhtQueryRecursionList(cid string, geolocation PeerGeolocation) DhtQueryLog {
	QueryLog := DhtQueryLog{
		Timestamp:        time.Now().Format(DateLayout),
		StartingCid:      cid,
		DhtRecursionList: make([]DhtQueryRecursionElem, 0),
	}

	queryResult := DhtQuery(cid)
	encounteredPeers := make(map[string]*Peer, 0)

	for _, row := range queryResult {

		// if we are facing a response from another peer
		if row["Type"] == float64(1) {

			senderCid := row["ID"].(string)
			senderPeer := addPeerToMap(senderCid, encounteredPeers)

			recursionElem := DhtQueryRecursionElem{
				Peer:     *senderPeer,
				PeerList: make([]Peer, 0),
			}

			//fetch Cid Response
			for _, response := range row["Responses"].([]interface{}) {
				responseCid := response.(map[string]interface{})["ID"].(string)
				responsePeer := addPeerToMap(responseCid, encounteredPeers)
				// fetch peer location
				for _, addr := range response.(map[string]interface{})["Addrs"].([]interface{}) {
					if responsePeer.Nation == "-" || responsePeer.Nation == "" {
						responsePeer.Addr = addr.(string)
						geolocation.SetPeerCity(responsePeer)
					}
				}

				//append peer elem to the list
				recursionElem.PeerList = append(recursionElem.PeerList, *responsePeer)
			}

			//append response log to the query log structure
			QueryLog.DhtRecursionList = append(QueryLog.DhtRecursionList, recursionElem)

		}
	}

	return QueryLog
}

// if we know this peer then fetch already obtained informations, otherwise, create it
func addPeerToMap(cid string, peerList map[string]*Peer) *Peer {
	sender := Peer{Cid: cid}
	// if we know this peer then fetch already obtained informations, otherwise, create it
	if val, ok := peerList[cid]; ok {
		sender = *val
	} else {
		peerList[cid] = &sender
	}
	return &sender
}
