package main

import (
	"net/http"
	"time"
	"os"
	"os/signal"
	"syscall"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
)

var (
	redisDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "redis_collection_duration_seconds",
		Help: "Duration of collections by the Redis-Exporter",
	}, []string{"module"},)

	redisRequestErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "redis_request_errors_total",
		Help: "Errors in requests to the Redis-Exporter",
	})

	test = prometheus.NewGauge(prometheus.GaugeOpts{
                Name: "test",
                Help: "test",
	})
)

func init() {
	prometheus.MustRegister(redisDuration)
	prometheus.MustRegister(redisRequestErrors)
	prometheus.MustRegister(version.NewCollector("redis_exporter"))
	prometheus.MustRegister(test)
}

func handler(w http.ResponseWriter, r *http.Request) {

	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "'target' parameter must be specified", 400)
		redisRequestErrors.Inc()
		return
	}

	object := r.URL.Query().Get("object")
	if object == "" {
		http.Error(w, "'object' parameter must be specified", 400)
		redisRequestErrors.Inc()
		return
	}

	log.Debugf("Scraping target '%s' with object '%s'", target, object)

	start := time.Now()
	registry := prometheus.NewRegistry()
	//test.Set(2)//collector(target)
	collector := collector{target: target, object: object}
	//registry.MustRegister(test)
	registry.MustRegister(collector)

	// Delegate http serving to Promethues client library, which will call collector.Collect.
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	duration := float64(time.Since(start).Seconds())
	log.Debugf("Scrape of target '%s' with module '%s' took %f seconds", target, duration)
}

func main() {
	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("Redis_Exporter : Beta"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting redis exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	hup := make(chan os.Signal)
	signal.Notify(hup, syscall.SIGHUP)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
            <head>
            <title>Redis-Exporter</title>
            <style>
            label{
            display:inline-block;
            width:75px;
            }
            form label {
            margin: 10px;
            }
            form input {
            margin: 10px;
            }
            </style>
            </head>
            <body>
            <h1>Redis-Exporter</h1>
            <form action="/redis">
            <label>Target:</label> <input type="text" name="target" placeholder="X.X.X.X" value="1.2.3.4"><br>
            <label>Object:</label> <input type="text" name="object" placeholder="object" value="Object Name"><br>
            <input type="submit" value="Submit">
            </form>
            </body>
            </html>`))
	})

	http.HandleFunc("/redis", handler)
	log.Infof("Listening on 9200")
	log.Fatal(http.ListenAndServe(":9200", nil))
}
