package lib

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// Url of the endpoint exposed for the ipfs swarm list api
// used to extract the list of all the node in the current swarm
const ipfsSwarmListHttpUrl = "http://127.0.0.1:5001/api/v0/swarm/peers?verbose=true"
const ipfsGetCidHttpUrl = "http://localhost:5001/api/v0/id"

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
