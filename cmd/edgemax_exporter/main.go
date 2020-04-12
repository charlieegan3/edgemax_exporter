// Command edgemax_exporter provides a Prometheus exporter for a Ubiquiti EdgeMAX
// Controller API and EdgeMAX devices.
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	edgemaxexporter "github.com/charlieegan3/edgemax_exporter"
	"github.com/charlieegan3/edgemax_exporter/edgemax"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// userAgent is ther user agent reported to the EdgeMAX Controller API.
	userAgent = "github.com/mdlayher/edgemax_exporter"
)

var (
	telemetryAddr = flag.String("telemetry.addr", ":9132", "host:port for EdgeMAX exporter")
	metricsPath   = flag.String("telemetry.path", "/metrics", "URL path for surfacing collected metrics")

	edgemaxAddr = flag.String("edgemax.addr", "", "address of EdgeMAX Controller API")
	username    = flag.String("edgemax.username", "", "username for authentication against EdgeMAX Controller API")
	password    = flag.String("edgemax.password", "", "password for authentication against EdgeMAX Controller API")

	insecure = flag.Bool("edgemax.insecure", false, "[optional] do not verify TLS certificate for EdgeMAX Controller API (warning: please use carefully)")
	timeout  = flag.Duration("edgemax.timeout", 5*time.Second, "[optional] timeout for EdgeMAX Controller API requests")
)

func main() {
	flag.Parse()

	if *edgemaxAddr == "" {
		log.Fatal("address of EdgeMAX Controller API must be specified with '-edgemax.addr' flag")
	}
	if *username == "" {
		log.Fatal("username to authenticate to EdgeMAX Controller API must be specified with '-edgemax.username' flag")
	}
	if *password == "" {
		log.Fatal("password to authenticate to EdgeMAX Controller API must be specified with '-edgemax.password' flag")
	}

	httpClient := &http.Client{Timeout: *timeout}
	if *insecure {
		httpClient = edgemax.InsecureHTTPClient(*timeout)
	}

	c, err := edgemax.NewClient(*edgemaxAddr, httpClient)
	if err != nil {
		log.Fatalf("cannot create EdgeMAX Controller client: %v", err)
	}
	c.UserAgent = userAgent

	if err := c.Login(*username, *password); err != nil {
		log.Fatalf("failed to authenticate to EdgeMAX Controller: %v", err)
	}

	e, done, err := edgemaxexporter.New(c)
	if err != nil {
		log.Fatalf("failed to create EdgeMAX exporter: %v", err)
	}
	defer done()

	prometheus.MustRegister(e)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, *metricsPath, http.StatusMovedPermanently)
	})

	log.Printf("Starting EdgeMAX exporter on %q for device at %q", *telemetryAddr, *edgemaxAddr)

	if err := http.ListenAndServe(*telemetryAddr, nil); err != nil {
		log.Fatalf("cannot start EdgeMAX exporter: %s", err)
	}
}
