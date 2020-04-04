package main

import (
	"github.com/giobart/IPFS-statistics-generator/lib"
	"github.com/ipfs/go-cid"
	"github.com/op/go-logging"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	DateLayout = "2006-01-02_15-04-05"
	ip42locdb  = "./IPV4-IP2LOCATION-LITE-DB3.BIN"
	ip6dbloc   = "./IPV6-IP2LOCATION-LITE-DB3.IPV6.BIN"
)

// How often the script must pull the statistics
var ticker = time.NewTicker(10 * time.Second)

// How often the script must plot the statistics
var plotTicker = time.NewTicker(30 * time.Minute)

// logger
var log = logging.MustGetLogger("go-ipfs-logger")

// Database
var database = lib.Database{}

// peer geolocalization class, used to set the peer location
var peerGolocalization lib.PeerGeolocation

/* Eevery n seconds -> pulls the statistics from the ipfs node. n="ticker" time */
func pullStatistics(stop <-chan bool, done chan<- bool) {

	for {
		select {
		case <-ticker.C:

			// collect peers from ipfs node
			peerList := lib.SwarmStatusList()
			cidList := make([]string, 0)

			for id, peer := range peerList {
				//check if peer has a valid addr
				if peer.Addr != "" {
					log.Info("[", id, "] - CID: [", peer.Cid, "] - Addr: [", peer.Addr, "] - Latency: [", peer.Latency, "]")

					c, err := cid.Decode(peer.Cid)
					if err != nil {
						log.Error("Error v1")
					}
					if c.Version() == 1 {
						log.Info("#### v1 ####")
					}
					log.Info("Got Cid: ", c.Bytes())
					// setting peer location
					peerGolocalization.SetPeerCity(&peer)

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

func main() {
	// channel for signal handling
	var sigs = make(chan os.Signal, 1)
	// channel to notify the function that is time to stop the execution
	var stop = make(chan bool, 1)
	// channel used to understand that function ended
	var done = make(chan bool)

	err := peerGolocalization.Init(ip42locdb, ip6dbloc)
	if err != nil {
		log.Error("No localization db connected")
		panic(1)
	}

	// db initialization
	database.DbInit()

	print(".-^-.-* Welcome to GO IPFS Analyzer *-.-^-. \n")
	print("Ctrl+c in any moment to quit the program \n")
	time.Sleep(time.Second * 3)

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
