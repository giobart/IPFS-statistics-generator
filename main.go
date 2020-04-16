package main

import (
	"github.com/giobart/IPFS-statistics-generator/lib"
	"github.com/op/go-logging"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	DateLayout = "2006-01-02_15-04-05"
	ip42locdb  = "./IPV4-IP2LOCATION-LITE-DB3.BIN"
	ip6dbloc   = "./IPV6-IP2LOCATION-LITE-DB3.IPV6.BIN"
)

// How often the script must pull the statistics
var statisticsTicker = time.NewTicker(30 * time.Second)

// logger
var log = logging.MustGetLogger("go-ipfs-logger")

// Database
var database = lib.Database{}

// peer geolocalization class, used to set the peer location
var peerGolocation lib.PeerGeolocation

/* Eevery n seconds -> pulls the statistics from the ipfs node. n="statisticsTicker" time */
func pullStatistics(stop <-chan bool, done chan<- bool) {

	for {
		select {
		case <-statisticsTicker.C:

			//pull generic swarm statistics
			pullPeerStatistics()

			//pull Dht buckets information
			pullDhtRecursionBucket()

		case <-stop:
			log.Info("## Stats pull Terminated ##")
			done <- true
			return
		}
	}

}

/*This function pulls from the ipfs node generic swarm statistics and save them to the database*/
func pullPeerStatistics() {
	// collect peers from ipfs node
	peerList := lib.SwarmStatusList()
	cidList := make([]string, 0)

	for id, peer := range peerList {
		//check if peer has a valid addr
		if peer.Addr != "" {
			log.Info("[", id, "] - CID: [", peer.Cid, "] - Addr: [", peer.Addr, "] - Latency: [", peer.Latency, "]")

			// setting peer location
			peerGolocation.SetPeerCity(&peer)

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
}

/*This function pulls from the ipfs node informations about
the dht by making recursive queries on the discovered buckets
this method implements PoW in order to discover new buckets*/
func pullDhtRecursionBucket() {

	myCid := lib.GetMyCid()

	// discover as many cid as possible in order to query the dht
	log.Info("Generating Buckets")
	buckets, err := lib.GenerateBucketQuery(100000, myCid)
	if err != nil {
		log.Error(err)
	}

	log.Info("Querying the dht with generated buckets")
	for i, bucket := range buckets {
		if bucket != "" {
			go func(i int, cid string) {
				//Query to the dht
				queryLog := lib.GetDhtQueryRecursionList(cid, peerGolocation)
				key := "bucket-" + strconv.Itoa(i)
				queryLog.BucketId = i
				//saving recursion to the db
				database.DbWrite("dhtQueryLog", key, queryLog)
				log.Info("Writing to DB the bucket: " + key)
			}(i, bucket)
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

	err := peerGolocation.Init(ip42locdb, ip6dbloc)
	if err != nil {
		log.Error("No localization db connected")
		panic(1)
	}

	// db initialization
	database.DbInit()

	print(".-^-.-* Welcome to GO IPFS Analyzer *-.-^-. \n")
	print("Ctrl+c in any moment to quit the program \n")
	time.Sleep(time.Second * 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// start pulling and plotting statistics
	go pullStatistics(stop, done)

	// serving statistics graphs
	lib.SetGraphDb(database)

	go lib.GraphsServe(":8081")

	// await for sigint or sigtem to stop application from pulling statistics
	select {
	//case keyboard interrupt
	case <-sigs:
		//sending 2 stop token for both pull and plot statistic function
		stop <- true
		//waiting for both functions to end
		<-done
		database.DbClose()
	}

	print(".-^-.-* Bye Bye *-.-^-. ")

}
