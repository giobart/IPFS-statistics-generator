package main

import (
	"encoding/json"
	"github.com/giobart/IPFS-statistics-generator/lib"
	"github.com/ip2location/ip2location-go"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/op/go-logging"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// How often the script must pull the statistics
var ticker = time.NewTicker(10 * time.Second)

// How often the script must plot the statistics
var plotTicker = time.NewTicker(30 * time.Minute)

// Url of the endpoint exposed for the ipfs swarm list api
// used to extract the list of all the node in the current swarm
var ipfsSwarmListHttpUrl = "http://127.0.0.1:5001/api/v0/swarm/peers?verbose=true"

// logger
var log = logging.MustGetLogger("go-ipfs-logger")

// Database
var database = lib.Database{}

//var ip location database
var ip42locdb, dbconnectionerroripv4 = ip2location.OpenDB("./IPV4-IP2LOCATION-LITE-DB3.BIN")

//var ip location database
var ip62locdb, dbconnectionerroripv6 = ip2location.OpenDB("./IPV6-IP2LOCATION-LITE-DB3.IPV6.BIN")

//date layout
var DateLayout = "2006-01-02_15-04-05"

/* Eevery n seconds -> pulls the statistics from the ipfs node. n="ticker" time */
func pullStatistics(stop <-chan bool, done chan<- bool) {

	for {
		select {
		case <-ticker.C:

			// collect peers from ipfs node
			peerList := swarmStatusList()
			cidList := make([]string, 0)

			for id, peer := range peerList {
				//check if peer has a valid addr
				if peer.Addr != "" {
					log.Info("[", id, "] - CID: [", peer.Cid, "] - Addr: [", peer.Addr, "] - Latency: [", peer.Latency, "]")

					// setting peer location
					setPeerCity(&peer)

					//storing peer info
					database.DbWrite("peers", peer.Cid, peer)

					//saving cid to cid list
					cidList = append(cidList, peer.Cid)
				}
			}
			//storing timestamp - peer list
			connection := lib.Connection{
				Timestamp: time.Now().Format(DateLayout),
				CidList:   cidList,
			}
			database.DbWrite("connections", connection.Timestamp, connection)
		case <-stop:
			log.Info("## Stats pull Terminated ##")
			done <- true
			return
		}
	}

}

/* Given a peer  -> set to the Peer the City and the Country from his ip address */
func setPeerCity(peer *lib.Peer) {

	var locdb *ip2location.DB
	ipAddr := ""

	// construct multiaddr from a string (err signals parse failure)
	multiaddr, err := ma.NewMultiaddr(peer.Addr)
	if err != nil {
		return
	}

	//swapping db between ipv6 or ipv4 to perform geolocation query
	for _, v := range multiaddr.Protocols() {
		if v.Code == ma.P_IP4 {
			locdb = ip42locdb
			ipAddr, _ = multiaddr.ValueForProtocol(v.Code)
			continue
		}
		if v.Code == ma.P_IP6 {
			locdb = ip62locdb
			ipAddr, _ = multiaddr.ValueForProtocol(v.Code)
			continue
		}

	}

	//fetching country and city from ipv4/ipv6 address db
	results, err := locdb.Get_all(ipAddr)
	if err != nil {
		log.Error(err)
	} else {
		peer.Nation = results.Country_short
		peer.City = results.City
	}

}

/* Every n seconds -> generate a plot of the current collected statistics. n= plotTicker seconds */
func plotStatistics(stop <-chan bool, done chan<- bool) {
	for {
		select {
		case <-plotTicker.C:
			//plot graphs from the previous pulled statistics
			lib.PlotStatistics(database)
		case <-stop:
			log.Info("## Plot Terminated ##")
			done <- true
			return
		}
	}
}

/* using ipfs HTTP api -> gets the list of the current connected peer to the swarm */
func swarmStatusList() []lib.Peer {
	var result = make([]lib.Peer, 1)

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
		p := lib.Peer{}
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
	database.DbInit()

	print(".-^-.-* Welcome to GO IPFS Analyzer *-.-^-. \n")
	print("Ctrl+c in any moment to quit the program \n")
	time.Sleep(time.Second * 3)

	// If Ip Location DB not detected then close program
	if dbconnectionerroripv4 != nil || dbconnectionerroripv6 != nil {
		panic(1)
	}

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// start pulling and plotting statistics
	go pullStatistics(stop, done)
	go plotStatistics(stop, done)

	// await for sigint or sigtem to stop application from pulling statistics
	select {
	//case keyboard interrupt
	case <-sigs:
		//sending 2 stop token for both pull and plot statistic function
		stop <- true
		stop <- true
		//waiting for both functions to end
		<-done
		<-done
		database.DbClose()
	}

	print(".-^-.-* Bye Bye *-.-^-. ")

}
