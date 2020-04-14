package lib

import (
	"github.com/go-echarts/go-echarts/charts"
	"github.com/mitchellh/mapstructure"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"
)

type Connections []Connection

//date layout
var DateLayout = "2006-01-02_15-04-05"

//database for statistics pump
var database Database

func (c Connections) Len() int {
	return len(c)
}

func (c Connections) Less(i, j int) bool {
	t1, _ := time.Parse(DateLayout, c[i].Timestamp)
	t2, _ := time.Parse(DateLayout, c[j].Timestamp)
	diff := t1.Sub(t2)
	if diff.Seconds() > 0 {
		return false
	} else {
		return true
	}
}

func (c Connections) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

/* Pie graph of the  peer nations */
func peerNationPie(peerMap map[string]interface{}, totPeer string) *charts.Pie {

	pie := charts.NewPie()
	pie.SetGlobalOptions(charts.TitleOpts{Title: "Peer By Nation", Subtitle: "peers: " + totPeer}, charts.LegendOpts{Top: "600px"})
	pie.Add("Peer nation", peerMap,
		charts.LabelTextOpts{Show: true, Formatter: "{b}: {c}"},
		charts.PieOpts{Radius: []string{"40%", "80%"}},
	)
	pie.Height = "600px"

	return pie
}

/* Map of the peers in the globe */
func peerNationMap(peerMap map[string]float32, maxPeer float32) *charts.Map {
	mc := charts.NewMap("world")
	mc.SetGlobalOptions(
		charts.TitleOpts{Title: "Peer Visual Map"},
		charts.VisualMapOpts{Calculable: true, Max: maxPeer + 10},
	)
	mc.Add("Peer distribution", peerMap)
	return mc
}

/* Average peer latency around the world */
func avgLatencyMap(peerLatencyMap map[string]float32, maxPeer float32) *charts.Map {

	mc := charts.NewMap("world")
	mc.SetGlobalOptions(
		charts.TitleOpts{Title: "Peer Latency Average (ms)"},
		charts.VisualMapOpts{Calculable: true, Max: maxPeer + 10},
		charts.InitOpts{Theme: charts.ThemeType.Infographic},
	)
	mc.Add("Latency (ms)", peerLatencyMap)
	return mc
}

/* Graph representing live connection numbers */
func peerConnectionsGraph(connections Connections, peers map[string]Peer) *charts.Line {

	//sort by date
	sort.Sort(connections)

	var stringDates = make([]string, 0)
	var peerList = make([]float32, 0)
	var peerChina = make([]float32, 0)
	var peerAmerica = make([]float32, 0)

	for _, v := range connections {
		stringDates = append(stringDates, v.Timestamp)
		peerList = append(peerList, float32(len(v.CidList)))
		countAmerica := 0
		countChina := 0
		//checking chinese and american peers
		for _, cid := range v.CidList {
			nation := peers[cid].Nation
			if nation == "United States" {
				countAmerica++
			}
			if nation == "China" {
				countChina++
			}
		}
		peerChina = append(peerChina, float32(countChina))
		peerAmerica = append(peerAmerica, float32(countAmerica))
	}

	kline := charts.NewLine()

	kline.AddXAxis(stringDates).AddYAxis("Total Connections", peerList)
	kline.AddXAxis(stringDates).AddYAxis("From China", peerChina)
	kline.AddXAxis(stringDates).AddYAxis("From America", peerAmerica)
	kline.SetGlobalOptions(
		charts.TitleOpts{Title: "Peer Connections "},
		charts.XAxisOpts{SplitNumber: 20},
		charts.YAxisOpts{Scale: true},
		charts.DataZoomOpts{XAxisIndex: []int{0}, Start: 50, End: 100},
	)
	return kline
}

/* Graph representing the node interconnection as a result of a dht query */
func graphRecursionDhtQuery(cid string, queryLog DhtQueryLog, peers map[string]Peer, id int) *charts.Graph {
	graph := charts.NewGraph()
	graph.SetGlobalOptions(charts.TitleOpts{Title: "Dht Query Recursion Graph"})
	myCid := GetMyCid()

	nodes := make([]charts.GraphNode, 0)
	links := make([]charts.GraphLink, 0)

	//generating nodes
	for _, n1 := range peers {
		node := charts.GraphNode{
			Name: n1.Nation + "-" + n1.City + "-" + n1.Cid,
			Y:    360 - (n1.Lat + 180),
			X:    (n1.Lon + 180),
		}
		nodes = append(nodes, node)
	}
	nodes = append(nodes, charts.GraphNode{
		Name:       "This Node",
		X:          180,
		Y:          160,
		SymbolSize: 20,
		ItemStyle:  charts.ItemStyleOpts{Color: "Blue"},
	})

	//generating links
	for i, n1 := range queryLog.DhtRecursionList {
		//connection with first 3 query nodes
		if i <= 2 {
			cidCompare(n1.Peer.Cid, cid)
			graphlink := charts.GraphLink{
				Source: "This Node",
				Target: n1.Peer.Nation + "-" + n1.Peer.City + "-" + n1.Peer.Cid,
				Value:  float32(cidCompare(n1.Peer.Cid, myCid)),
			}
			links = append(links, graphlink)
		}
		for _, link := range n1.PeerList {
			graphlink := charts.GraphLink{
				Source: n1.Peer.Nation + "-" + n1.Peer.City + "-" + n1.Peer.Cid,
				Target: link.Nation + "-" + link.City + "-" + link.Cid,
				Value:  float32(cidCompare(link.Cid, myCid)),
			}
			links = append(links, graphlink)
		}
	}

	graph.SetGlobalOptions(charts.TitleOpts{
		Title:         "Recursion graph for bucket: " + strconv.Itoa(id),
		TitleStyle:    charts.TextStyleOpts{},
		Subtitle:      "Starting Cid: " + cid,
		SubtitleStyle: charts.TextStyleOpts{},
	})
	graph.Add("Recursion graph for bucket: "+strconv.Itoa(id), nodes, links,
		charts.GraphOpts{Layout: "none", FocusNodeAdjacency: true, Roam: true},
		charts.EmphasisOpts{Label: charts.LabelTextOpts{Show: false, Position: "left", Color: "black"}},
		charts.LineStyleOpts{Curveness: 0.3},
	)

	return graph
}

func SetGraphDb(db Database) {
	database = db
}

/* handler that prepare the data and build the /stats web page */
func peerHandler(w http.ResponseWriter, _ *http.Request) {

	log.Info("Extracting peers from DB")
	list := database.dbReadAll("peers")
	peerList := make(map[string]Peer, 0)
	peerMap := make(map[string]interface{})
	peerMapFloat := make(map[string]float32)
	peerAvgLatency := make(map[string]float32)
	maxPeer := float32(0)
	maxLatency := float32(0)
	maxLatencyCity := ""

	for _, el := range list {
		if el != nil {
			peer := Peer{}
			if err := mapstructure.Decode(el, &peer); err != nil {
				log.Error(err)
			} else {
				if peer.Nation == "United States of America" {
					peer.Nation = "United States"
				}
				if peer.Nation == "Russian Federation" {
					peer.Nation = "Russia"
				}
				if peer.Nation == "United Kingdom of Great Britain and Northern Ireland" {
					peer.Nation = "United Kingdom"
				}
				if len(peer.Nation) >= 2 {
					peerList[peer.Cid] = peer
				}
			}
		}
	}

	//generation of all peer map structure needed
	for _, p := range peerList {
		if content, present := peerMap[p.Nation]; present {
			oldPeerNum := content.(float32)
			newPeerNum := oldPeerNum + float32(1)
			peerMap[p.Nation] = newPeerNum
			peerMapFloat[p.Nation] = newPeerNum

			if content.(float32) > maxPeer {
				maxPeer = content.(float32)
			}

			//calculating average latency
			if len(p.Latency) > 2 {
				multiplier := 1
				val, err := strconv.ParseFloat(p.Latency[:len(p.Latency)-2], 32)
				//if time is in seconds and not in milliseconds, multiply by 1000
				if p.Latency[len(p.Latency)-2:] != "ms" {
					multiplier = 1000
				}
				if err == nil {
					val = float64(multiplier) * val
					//updating actual average latency
					peerAvgLatency[p.Nation] = (peerAvgLatency[p.Nation]*oldPeerNum + float32(val)) / newPeerNum
					if peerAvgLatency[p.Nation] > maxLatency || maxLatencyCity == p.Nation {
						maxLatency = peerAvgLatency[p.Nation]
						maxLatencyCity = p.Nation
					}
				}
			}

		} else {
			peerMap[p.Nation] = float32(1.0)
			peerMapFloat[p.Nation] = float32(1.0)
			val, err := strconv.ParseFloat(p.Latency, 32)
			if err == nil {
				peerAvgLatency[p.Nation] = float32(val)
			}

		}
	}

	log.Info("Extracting connections from DB")
	connections := database.dbReadAll("connections")
	connectionList := make(Connections, 0)

	for _, el := range connections {
		if el != nil {
			conn := Connection{}
			if err := mapstructure.Decode(el, &conn); err != nil {
				log.Error(err)
			} else {
				connectionList = append(connectionList, conn)
			}
		}
	}

	page := charts.NewPage()
	page.Add(
		peerNationPie(peerMap, strconv.Itoa(len(list))),
		peerNationMap(peerMapFloat, maxPeer),
		avgLatencyMap(peerAvgLatency, maxLatency),
		peerConnectionsGraph(connectionList, peerList),
	)
	f, err := os.Create("stats.html")
	if err != nil {
		log.Error(err)
	}
	_ = page.Render(w, f)
}

/* handler that prepare data and build the /worldGraph page */
func worldGraphHandler(w http.ResponseWriter, _ *http.Request) {

	queryLogResults := database.dbReadAll("dhtQueryLog")
	var queryBuckets [256]DhtQueryLog
	peersMap := make([]map[string]Peer, 256)

	//parsing db entry
	for _, el := range queryLogResults {
		queryElem := DhtQueryLog{}
		if err := mapstructure.Decode(el, &queryElem); err != nil {
			log.Error(err)
		} else {
			//parsing query bucket
			queryBuckets[queryElem.BucketId] = queryElem
			peersMap[queryElem.BucketId] = make(map[string]Peer)
			//adding involved peers to peer map
			for _, p := range queryElem.DhtRecursionList {
				peersMap[queryElem.BucketId][p.Peer.Cid] = p.Peer
				for _, p2 := range p.PeerList {
					peersMap[queryElem.BucketId][p2.Cid] = p2
				}
			}
		}
	}

	//generate graph in the page
	page := charts.NewPage()
	for i, bucket := range queryBuckets {
		if bucket.StartingCid != "" {
			page.Add(graphRecursionDhtQuery(bucket.StartingCid, bucket, peersMap[i], i))
		}
	}

	f, err := os.Create("recursion.html")
	if err != nil {
		log.Error(err)
	}
	page.Render(w, f)
}

func GraphsServe(port string) {
	http.HandleFunc("/stats", peerHandler)
	http.HandleFunc("/worldGraph", worldGraphHandler)
	http.ListenAndServe(port, nil)
}
