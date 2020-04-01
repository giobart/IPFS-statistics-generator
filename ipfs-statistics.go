package main

import (
	"encoding/json"
	"fmt"
	"github.com/ip2location/ip2location-go"
	"github.com/op/go-logging"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// struct reperesenting the peer
type Peer struct {
	Addr    string `json:"Addr"`
	Cid     string `json:"Cid"`
	Latency string `json:"Latency"`
	Nation  string `json:"Nation"`
	City    string `json:"City"`
}

type Connection struct {
	Timestamp string   `json:"timestamp"`
	CidList   []string `json:"cidList"`
}

// How often the script must pull the statistics
var ticker = time.NewTicker(1 * time.Second)

// Url of the endpoint exposed for the ipfs swarm list api
// used to extract the list of all the node in the current swarm
var ipfsSwarmListHttpUrl = "http://127.0.0.1:5001/api/v0/swarm/peers"

// logger
var log = logging.MustGetLogger("go-ipfs-logger")

// Database
var database = Database{}

//var ip location database
var ipdb, dbconnectionerror = ip2location.OpenDB("./IP2LOCATION-LITE-DB3.BIN")

//date layout
var DateLayout = "2006-01-02_15-04-05"

// periodically pulls the statistics from the ipfs node
func pullStatistics(stop <-chan bool, done chan<- bool) {

	for {
		select {
		case <-stop:
			log.Info("## Terminated ##")
			done <- true
			return
		case <-ticker.C:
			peerList := swarmStatusList()
			cidList := make([]string, 0)
			for id, peer := range peerList {
				//check if peer has a valid addr
				if peer.Addr != "" {
					log.Info("[", id, "] - CID: [", peer.Cid, "] - Addr: [", peer.Addr, "] - Latency: [", peer.Latency, "]")

					//fetching country and city from ip4 address
					results, err := ipdb.Get_all(strings.Split(peer.Addr, "/")[2])
					if err != nil {
						log.Error(err)
					} else {
						peer.Nation = results.Country_short
						peer.City = results.City
					}

					//storing peer info
					database.dbWrite("peers", peer.Cid, peer)

					//saving cid to cid list
					cidList = append(cidList, peer.Cid)
				}
			}
			//storing timestamp - peer list
			connection := Connection{
				Timestamp: time.Now().Format(DateLayout),
				CidList:   cidList,
			}
			database.dbWrite("connections", connection.Timestamp, connection)
		}
	}

}

// using ipfs HTTP api gets the list of the current connected peer to the swarm
func swarmStatusList() []Peer {
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

func main() {
	// channel for signal handling
	var sigs = make(chan os.Signal, 1)
	// channel to notify the function that is time to stop the execution
	var stop = make(chan bool, 1)
	// channel used to understand that function ended
	var done = make(chan bool)

	// db initialization
	database.dbInit()

	print(".-^-.-* Welcome to GO IPFS Analyzer *-.-^-. \n")
	print("digit 1 - to start analyzing your ipfs node \n")
	print("digit 2 - to plot collected statistics \n")
	print("Ctrl+c in any moment to quit the program \n")

	var digit int
	_, err := fmt.Scanf("%d", &digit)
	if err != nil {
		log.Error(err)
		panic(1)
	}

	switch digit {
	//start pulling statistics from ipfs
	case 1:
		// Ip Location database
		if dbconnectionerror != nil {
			panic(1)
		}

		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go pullStatistics(stop, done)

		// await for sigint or sigtem to stop application from pulling statistics
		select {
		case <-sigs:
			stop <- true
			<-done
			database.dbClose()
		}
	//plot graphs from the previous pulled statistics
	case 2:
		plotStatistics()
	default:
		print("#### Use a correct digit ####")
	}

}
