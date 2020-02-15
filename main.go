package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/expfmt"
)

const (
	defaultListenAddr = "127.0.0.1:9375"
	defaultInterval   = 10
	GPLv2             = "https://www.ohse.de/uwe/licenses/GPL-2"
)

type ServerEntry struct {
	Server_IP     string
	Server_number int
	Server_name   string
	Product       string
	Dc            string
	Traffic       string
	Flatrate      bool
	Status        string
	Throttled     bool
	Canceled      bool
	Paid_until    string
	IP            []string
	Subnet        []struct {
		IP   string
		Mask string
	}
}
type ServerList []struct {
	Server ServerEntry
}

type Traffic struct {
	Traffic struct {
		Type string
		From string
		To   string
		Data map[string]struct {
			In  float64
			Out float64
			Sum float64
		}
	}
}

type APIError struct {
	Error struct {
		Status int    `json:"status"`
		Code   string `json:"code"`
	} `json:"error"`
}

type TrafficInfo struct {
	address       string
	input         float64
	output        float64
	total         float64
	server_number int
	server_name   string
	dns_name      string
	product       string
}

var (
	hetznerUsername string
	hetznerPassword string
	labels          = []string{"address", "dns_name", "server_name", "server_number", "product"}
	inputGB         = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "hetzner_traffic",
			Name:      "input_gb",
			Help:      "Input traffic in GB",
		},
		labels,
	)
	outputGB = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "hetzner_traffic",
			Name:      "output_gb",
			Help:      "Output traffic in GB",
		},
		labels,
	)
	totalGB = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "hetzner_traffic",
			Name:      "total_gb",
			Help:      "Used total traffic (input and output) in GB",
		},
		labels,
	)
	flagOneshot    = flag.Bool("1", false, "collect and output the metrics once, and exit.")
	flagVersion    = flag.Bool("version", false, "show version information and exit.")
	flagLicense    = flag.Bool("license", false, "show license information and exit.")
	flagLogUpdates = flag.Bool("log-updates", false, "log updates.")
	flagInterval   = flag.Int("interval", defaultInterval, "run updates against the API every ... minutes.")
	flagListen     = flag.String("listen", defaultListenAddr,
		"Address on which to expose metrics and web interface.")
)

func basicRequest(client *http.Client, method, apiurl string, data io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, apiurl, data)
	if err != nil {
		log.Fatal(err) // broken URL, bad method, can't continue.
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(hetznerUsername, hetznerPassword)
	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}

	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	if resp.StatusCode != 200 {
		var apiErr APIError
		err = json.Unmarshal(bodyText, &apiErr)
		if err != nil {
			return []byte{}, err
		}
		return []byte{},
			fmt.Errorf("API error: %d - %v", apiErr.Error.Status, apiErr.Error.Code)
	}
	return bodyText, nil
}
func handleRDNS(client *http.Client) (map[string]string, error) {
	out := make(map[string]string)
	bodyText, err := basicRequest(client, "GET", "https://robot-ws.your-server.de/rdns", nil)
	if err != nil {
		return out, err
	}

	var t []struct {
		Rdns struct {
			IP  string
			Ptr string
		}
	}

	err = json.Unmarshal(bodyText, &t)
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range t {
		out[entry.Rdns.IP] = entry.Rdns.Ptr
	}
	return out, nil
}
func getTraffic(client *http.Client, par url.Values) (Traffic, error) {
	var trafficresponse Traffic
	bodyText, err := basicRequest(client, "POST", "https://robot-ws.your-server.de/traffic",
		strings.NewReader(par.Encode()))
	if err != nil {
		return trafficresponse, err
	}
	err = json.Unmarshal(bodyText, &trafficresponse)
	return trafficresponse, err
}

func updateIPs() ([]TrafficInfo, error) {
	client := &http.Client{}

	out := make([]TrafficInfo, 0)

	/* part1: get the server list */
	bodyText, err := basicRequest(client, "GET", "https://robot-ws.your-server.de/server", nil)
	if err != nil {
		return out, err
	}

	stich := time.Now()
	// hetzner returns the traffic after the hour is finished. we avoid data
	// loss by going back one hour.
	stich = stich.Add(time.Hour * -1)
	stichString := stich.Format("2006-01")

	var serverlistresponse ServerList
	err = json.Unmarshal(bodyText, &serverlistresponse)
	if err != nil {
		return out, err
	}

	/* build back link list and params for part3 */

	ipToServer := make(map[string]ServerEntry)
	par := url.Values{}
	par.Set("type", "month")
	par.Set("from", stichString+"-01")
	par.Set("to", stichString+"-31")
	for _, entry := range serverlistresponse {
		for _, ip := range entry.Server.IP {
			ipToServer[ip] = entry.Server
			par.Add("ip[]", ip)
		}
	}

	for _, entry := range serverlistresponse {
		for _, sub := range entry.Server.Subnet {
			t := sub.IP + "/" + sub.Mask
			ipToServer[t] = entry.Server
			par.Add("subnet[]", sub.IP)
		}
	}

	/* part2: get the revdns list */
	rdns, err := handleRDNS(client)
	if err != nil {
		return out, err
	}

	/* part3: get the traffic */
	trafficresponse, err := getTraffic(client, par)
	if err != nil {
		return out, err
	}
	for key, entry := range trafficresponse.Traffic.Data {
		var ti TrafficInfo
		ti.address = key
		ti.input = entry.In
		ti.output = entry.Out
		ti.total = entry.Sum

		s, ok := ipToServer[key]
		if ok {
			ti.server_number = s.Server_number
			ti.server_name = s.Server_name
			ti.product = s.Product
		}
		r, ok := rdns[key]
		if !ok {
			tmp := strings.Split(key, "/")
			r, ok = rdns[tmp[0]]
		}
		if ok {
			ti.dns_name = r
		}
		out = append(out, ti)
	}
	return out, nil
}

func updateMetrics(oneshot bool) {
	for {
		start := time.Now()
		if *flagLogUpdates {
			log.Printf("update starts\n")
		}
		tiList, err := updateIPs()
		if err != nil {
			log.Printf("update failed: %s\n", err)
			continue
		}
		end := time.Now()
		inputGB.Reset()
		outputGB.Reset()
		totalGB.Reset()
		var curTotal float64 = 0.0
		for _, ti := range tiList {
			inputGB.With(prometheus.Labels{
				"address":       ti.address,
				"server_number": strconv.Itoa(ti.server_number),
				"server_name":   ti.server_name,
				"dns_name":      ti.dns_name,
				"product":       ti.product,
			}).Add(ti.input)
			outputGB.With(prometheus.Labels{
				"address":       ti.address,
				"server_number": strconv.Itoa(ti.server_number),
				"server_name":   ti.server_name,
				"dns_name":      ti.dns_name,
				"product":       ti.product,
			}).Add(ti.output)
			totalGB.With(prometheus.Labels{
				"address":       ti.address,
				"server_number": strconv.Itoa(ti.server_number),
				"server_name":   ti.server_name,
				"dns_name":      ti.dns_name,
				"product":       ti.product,
			}).Add(ti.total)
			curTotal += ti.total
		}
		if *flagLogUpdates {
			d := end.Sub(start)
			log.Printf("update ended: total=%v, dur=%v\n", curTotal, d)
		}

		if oneshot {
			return
		}
		i := *flagInterval
		if i < 1 {
			i = 1
		}
		time.Sleep(time.Duration(i) * 60 * time.Second)
	}
}

func handleOneshot() {
	updateMetrics(true)
	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,
	}
	gathering, err := gatherers.Gather()
	if err != nil {
		log.Fatalf("Gather failed: %v\n", err)
	}

	for _, mf := range gathering {
		_, err := expfmt.MetricFamilyToText(os.Stdout, mf)
		if err != nil {
			log.Fatalf("Export failed: %v\n", err)
		}
	}
}

func main() {
	flag.Parse()

	if *flagVersion {
		fmt.Printf("%s: version %s\n", os.Args[0], versionString)
		os.Exit(0)
	}
	if *flagLicense {
		fmt.Printf("%s: version %s\n\nThis software is published under the terms of the GPL version 2.\nA copy is at %s.\n",
			os.Args[0], versionString, GPLv2)
		os.Exit(0)
	}

	hetznerUsername = os.Getenv("HETZNER_USER")
	hetznerPassword = os.Getenv("HETZNER_PASS")
	if hetznerUsername == "" || hetznerPassword == "" {
		log.Fatal("Please provide HETZNER_USER and HETZNER_PASS as environment variables")
	}

	prometheus.MustRegister(inputGB)
	prometheus.MustRegister(outputGB)
	prometheus.MustRegister(totalGB)
	if *flagOneshot {
		handleOneshot()
		os.Exit(0)
	}

	go updateMetrics(false)

	fmt.Printf("Listening on %q\n", *flagListen)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*flagListen, nil))
}
