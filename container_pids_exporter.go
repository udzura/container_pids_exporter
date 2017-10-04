package main

import (
	"net/http"
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "conainer_pids"
)

var (
	up = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Was the last query of pids check successful.",
		nil, nil,
	)
)

// Exporter collects Consul stats from the given server and exports them using
// the prometheus metrics package.
type Exporter struct {
	filter *regexp.Regexp
}

func NewExporter() (*Exporter, error) {
	return &Exporter{}, nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	// We'll use peers to decide that we're up.
	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, 1,
	)
}

func init() {
	prometheus.MustRegister(version.NewCollector("container_pids_exporter"))
}

func main() {
	var (
		listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":8099").String()
	)
	kingpin.Version(version.Print("container_pids_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	log.Infoln("Starting container_pids_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	exporter, err := NewExporter()
	if err != nil {
		log.Fatalln(err)
	}

	prometheus.MustRegister(exporter)

	http.Handle("/metrics", prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`Container's pids exporter.
Please visit /metrics !!`))
	})

	log.Infoln("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}