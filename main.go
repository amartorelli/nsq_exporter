package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Client struct {
	ClientID      string `json:"client_id"`
	Hostname      string `json:"hostname"`
	Version       string `json:"version"`
	RemoteAddr    string `json:"remote_address"`
	ReadyCount    int    `json:"ready_count"`
	InFlightCount int    `json:"in_flight_count"`
	MessageCount  int    `json:"message_count"`
	FinishCount   int    `json:"finish_count"`
	RequeueCount  int    `json:"requeue_count"`
}

type Channel struct {
	ChannelName   string   `json:"channel_name"`
	Depth         int      `json:"depth"`
	BackendDepth  int      `json:"backend_depth"`
	InFlightCount int      `json:"in_flight_count"`
	DeferredCount int      `json:"deferred_count"`
	MessageCount  int      `json:"message_count"`
	RequeueCount  int      `json:"requeue_count"`
	TimeoutCount  int      `json:"timeout_count"`
	ClientCount   int      `json:"client_count"`
	Clients       []Client `json:"clients"`
	Paused        bool     `json:"paused"`
}

type Topic struct {
	TopicName string    `json:"topic_name"`
	Channels  []Channel `json:"channels"`
}

type Stats struct {
	Version string  `json:"version"`
	Topics  []Topic `json:"topics"`
}

type nsqCollector struct {
	namespace          string
	clientCountGauge   *prometheus.GaugeVec
	messageCountGauge  *prometheus.GaugeVec
	depthGauge         *prometheus.GaugeVec
	inFlightCountGauge *prometheus.GaugeVec
}

func NewNSQCollector(namespace string) *nsqCollector {
	return &nsqCollector{
		namespace: namespace,
		clientCountGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "client_count",
				Help:      "Number of clients connected to the channel",
			},
			[]string{"topic", "channel", "paused"},
		),
		messageCountGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "message_count",
				Help:      "Number of messages in the channel",
			},
			[]string{"topic", "channel", "paused"},
		),
		depthGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "depth",
				Help:      "Depth of the channel's queue",
			},
			[]string{"topic", "channel", "paused"},
		),
		inFlightCountGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "in_flight_count",
				Help:      "Number of messages currently in-flight in the channel",
			},
			[]string{"topic", "channel", "paused"},
		),
	}
}

func (c *nsqCollector) Describe(ch chan<- *prometheus.Desc) {
	c.clientCountGauge.Describe(ch)
	c.messageCountGauge.Describe(ch)
	c.depthGauge.Describe(ch)
	c.inFlightCountGauge.Describe(ch)
}

func (c *nsqCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := c.fetchStats()
	if err != nil {
		log.Println("Error fetching stats:", err)
		return
	}

	for _, topic := range stats.Topics {
		for _, channel := range topic.Channels {
			labels := prometheus.Labels{
				"topic":   topic.TopicName,
				"channel": channel.ChannelName,
				"paused":  strconv.FormatBool(channel.Paused),
			}

			// Set gauge values
			c.clientCountGauge.With(labels).Set(float64(channel.ClientCount))
			c.messageCountGauge.With(labels).Set(float64(channel.MessageCount))
			c.depthGauge.With(labels).Set(float64(channel.Depth))
			c.inFlightCountGauge.With(labels).Set(float64(channel.InFlightCount))
		}
	}

	// Collect the metrics
	c.clientCountGauge.Collect(ch)
	c.messageCountGauge.Collect(ch)
	c.depthGauge.Collect(ch)
	c.inFlightCountGauge.Collect(ch)
}

var (
	listenAddress = flag.String("web.listen", ":9117", "Address on which to expose metrics and web interface.")
	metricsPath   = flag.String("web.path", "/metrics", "Path under which to expose metrics.")
	nsqdURL       = flag.String("nsqd.addr", "http://localhost:4151/stats", "Address of the nsqd node.")
)

func (c *nsqCollector) fetchStats() (*Stats, error) {
	resp, err := http.Get(fmt.Sprintf("%s?format=json", *nsqdURL))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %v", err)
	}
	defer resp.Body.Close()

	var stats Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode stats JSON: %v", err)
	}

	return &stats, nil
}

func main() {
	namespace := "nsq"

	// Create a new NSQ collector
	collector := NewNSQCollector(namespace)

	// Register the collector with Prometheus
	prometheus.MustRegister(collector)

	// Expose the metrics at /metrics using the updated HandlerFor function
	http.Handle(*metricsPath, promhttp.Handler())
	if *metricsPath != "" && *metricsPath != "/" {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html>
			<head><title>NSQ Exporter</title></head>
			<body>
			<h1>NSQ Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
		})
	}
	log.Printf("Listening on %s\n", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
