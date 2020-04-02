package lib

import (
	"github.com/mitchellh/mapstructure"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
	"os"
	"sort"
	"time"
)

type Connections []Connection

//date layout
var DateLayout = "2006-01-02_15-04-05"

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

func generateNationsChart(peers []Peer) {

	nations := make(map[string]int)

	for _, p := range peers {
		if content, present := nations[p.Nation]; present {
			nations[p.Nation] = content + 1
		} else {
			nations[p.Nation] = 1
		}
	}

	vals := make([]chart.Value, 0)
	for key, val := range nations {
		vals = append(vals, chart.Value{Label: key, Value: float64(val)})
	}

	graph := chart.BarChart{
		Title: "Peer swarm nations",
		Background: chart.Style{
			Padding: chart.Box{
				Top: 40,
			},
		},
		Height:   1000,
		BarWidth: 30,
		Bars:     vals,
	}

	f, _ := os.Create("graphs/nations.png")
	defer f.Close()
	_ = graph.Render(chart.PNG, f)

}

func generateConnectionChart(connectionList []Connection) {

	var stringDates = make([]string, 0)
	var peerList = make([]float64, 0)
	var max = 0
	for _, v := range connectionList {
		stringDates = append(stringDates, v.Timestamp)
		peerList = append(peerList, float64(len(v.CidList)))
		if len(v.CidList) > max {
			max = len(v.CidList)
		}
	}

	var dates []time.Time
	for _, ts := range stringDates {
		parsed, _ := time.Parse(DateLayout, ts)
		dates = append(dates, parsed)
	}
	xv := dates
	yv := peerList

	priceSeries := chart.TimeSeries{
		Name: "SPY",
		Style: chart.Style{
			StrokeColor: chart.GetDefaultColor(0),
			Show:        true,
		},
		XValues: xv,
		YValues: yv,
	}

	smaSeries := chart.SMASeries{
		Name: "SPY - SMA",
		Style: chart.Style{
			StrokeColor:     drawing.ColorRed,
			StrokeDashArray: []float64{5.0, 5.0},
			Show:            true,
		},
		InnerSeries: priceSeries,
	}

	bbSeries := &chart.BollingerBandsSeries{
		Name: "SPY - Bol. Bands",
		Style: chart.Style{
			StrokeColor: drawing.ColorFromHex("efefef"),
			FillColor:   drawing.ColorFromHex("efefef").WithAlpha(64),
			Show:        true,
		},
		InnerSeries: priceSeries,
	}

	graph := chart.Chart{
		XAxis: chart.XAxis{
			TickPosition: chart.TickPositionBetweenTicks,
		},
		YAxis: chart.YAxis{
			Range: &chart.ContinuousRange{
				Max: float64(max),
				Min: 0.0,
			},
		},
		Series: []chart.Series{
			bbSeries,
			priceSeries,
			smaSeries,
		},
	}

	f, _ := os.Create("graphs/connection-time.png")
	defer f.Close()
	err := graph.Render(chart.PNG, f)
	if err != nil {
		log.Error(err)
	}
}

func PlotStatistics(database Database) {
	//pushing in memory peer list
	log.Info("Extracting peers from DB")
	list := database.dbReadAll("peers")
	peerList := make([]Peer, 0)

	for _, el := range list {
		if el != nil {
			peer := Peer{}
			if err := mapstructure.Decode(el, &peer); err != nil {
				log.Error(err)
			} else {
				peerList = append(peerList, peer)
			}
		}
	}

	//pushing in memory connection list
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
	sort.Sort(connectionList)

	//plotting nations graph
	log.Info("Generating nations graph")
	generateNationsChart(peerList)

	//plotting peer number by time
	generateConnectionChart(connectionList)

	//plotting peer avg by time
	//TODO

}
