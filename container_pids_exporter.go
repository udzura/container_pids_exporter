package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

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
	max = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "max"),
		"Current pids.max value of the container.",
		[]string{"id"}, nil,
	)
	current = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "current"),
		"Current pids.current value of the container.",
		[]string{"id"}, nil,
	)
)

// Exporter collects Consul stats from the given server and exports them using
// the prometheus metrics package.
type Exporter struct {
	filter     *regexp.Regexp
	cgroupRoot string
}

func NewExporter(root string) (*Exporter, error) {
	return &Exporter{cgroupRoot: root}, nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- max
	ch <- current
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	// We'll use peers to decide that we're up.
	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, 1,
	)

	err := filepath.Walk(
		e.cgroupRoot+"/pids",
		func(dir string, info os.FileInfo, err error) error {
			if info.IsDir() {
				log.Debugf("Inspecting: %v", dir)
				var value []byte
				var tmp string
				var v float64
				var err error
				containerID := "/" + path.Base(dir)
				// skip when open failed...
				if value, err = ioutil.ReadFile(dir + "/pids.max"); err != nil {
					log.Warnf("Something is wrong on Collect: %v", err)
					return nil
				}
				if strings.HasPrefix(string(value), "max") {
					ch <- prometheus.MustNewConstMetric(
						max, prometheus.GaugeValue, float64(-1), containerID,
					)
				} else {
					tmp = strings.TrimSpace(string(value))
					if v, err = strconv.ParseFloat(tmp, 64); err != nil {
						log.Warnf("Something is wrong on Collect: %v", err)
						return nil
					}
					ch <- prometheus.MustNewConstMetric(
						max, prometheus.GaugeValue, v, containerID,
					)
				}

				if value, err = ioutil.ReadFile(dir + "/pids.current"); err != nil {
					log.Warnf("Something is wrong on Collect: %v", err)
					return nil
				}
				tmp = strings.TrimSpace(string(value))
				if v, err = strconv.ParseFloat(tmp, 64); err != nil {
					log.Warnf("Something is wrong on Collect: %v", err)
					return nil
				}
				ch <- prometheus.MustNewConstMetric(
					current, prometheus.GaugeValue, v, containerID,
				)
			}
			return nil
		},
	)
	if err != nil {
		log.Warnf("Something is wrong on Collect: %v", err)
	}
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

	exporter, err := NewExporter("/sys/fs/cgroup")
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
