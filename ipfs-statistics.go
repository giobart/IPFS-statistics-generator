package main

import (
	"encoding/json"
	"github.com/op/go-logging"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Peer struct {
	addr    string `json:"Addr"`
	cid     string `json:"Peer"`
	latency string `json:"Latency"`
}

// How often the script must pull the statistics
var ticker = time.NewTicker(1 * time.Second)

// Url of the endpoint exposed for the ipfs swarm list api
// used to extract the list of all the node in the current swarm
var ipfs_swarm_list_http_url = "http://127.0.0.1:5001/api/v0/swarm/peers"

// logger
var log = logging.MustGetLogger("go-ipfs-logger")

// periodically pulls the statistics from the ipfs node
func pull_statistics(stop <-chan bool, done chan<- bool) {

	for {
		select {
		case <-stop:
			log.Info("## Terminated ##")
			done <- true
			return
		case <-ticker.C:
			peerList := swarm_status_list()
			for id, peer := range peerList {
				log.Info("[", id, "] - CID: [", peer.cid, "] - Addr: [", peer.addr, "]")
			}

		}
	}

}

// using ipfs HTTP api gets the list of the current connected peer to the swarm
func swarm_status_list() []Peer {
	var result = make([]Peer, 1)

	//http request to ipfs api
	resp, err := http.Get(ipfs_swarm_list_http_url)
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
		p.addr = currPeer["Addr"].(string)
		p.cid = currPeer["Peer"].(string)
		p.latency = currPeer["Latency"].(string)
		result = append(result, p)
	}

	return result
}

func main() {
	// channel for signal handling
	var sigs = make(chan os.Signal, 1)
	// channel to notify the function that is time to stop the execution
	var stop = make(chan bool, 1)
	// channel used to understand that function ended
	var done = make(chan bool)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go pull_statistics(stop, done)

	// await for sigint or sigtem to stop application from pulling statistics
	select {
	case <-sigs:
		stop <- true
		<-done
	}

}
