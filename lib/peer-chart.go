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

func peerNationMap(peerMap map[string]float32, maxPeer float32) *charts.Map {
	mc := charts.NewMap("world")
	mc.SetGlobalOptions(
		charts.TitleOpts{Title: "Peer Visual Map"},
		charts.VisualMapOpts{Calculable: true, Max: maxPeer + 10},
	)
	mc.Add("map", peerMap)
	return mc
}

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

func SetGraphDb(db Database) {
	database = db
}

func pieHandler(w http.ResponseWriter, _ *http.Request) {

	log.Info("Extracting peers from DB")
	list := database.dbReadAll("peers")
	peerList := make(map[string]Peer, 0)
	peerMap := make(map[string]interface{})
	peerMapFloat := make(map[string]float32)
	maxPeer := float32(0)

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

	for _, p := range peerList {
		if content, present := peerMap[p.Nation]; present {
			peerMap[p.Nation] = content.(float32) + float32(1)
			peerMapFloat[p.Nation] = content.(float32) + float32(1)
			if content.(float32) > maxPeer {
				maxPeer = content.(float32)
			}
		} else {
			peerMap[p.Nation] = float32(1.0)
			peerMapFloat[p.Nation] = float32(1.0)
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
		peerConnectionsGraph(connectionList, peerList),
	)
	f, err := os.Create("stats.html")
	if err != nil {
		log.Error(err)
	}
	_ = page.Render(w, f)
}

func GraphsServe(port string) {
	http.HandleFunc("/", pieHandler)
	http.ListenAndServe(port, nil)
}
