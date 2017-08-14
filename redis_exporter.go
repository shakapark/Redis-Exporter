package main

import (
//	"fmt"
	"net/http"
	"time"

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

	sc = &SafeConfig{
		C: &config.Config{},
	}
	reloadCh chan chan error
)

type SafeConfig struct {
	sync.RWMutex
	C *config.Config
}

func init() {
	prometheus.MustRegister(redisDuration)
	prometheus.MustRegister(redisRequestErrors)
	prometheus.MustRegister(version.NewCollector("redis_exporter"))
}

func handler(w http.ResponseWriter, r *http.Request) {

	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "'target' parameter must be specified", 400)
		redisRequestErrors.Inc()
		return
	}

	moduleName := r.URL.Query().Get("module")
	if moduleName == "" {
		moduleName = "default"
	}

	sc.RLock()
	module, ok := (*(sc.C))[moduleName]
	sc.RUnlock()
	if !ok {
		http.Error(w, fmt.Sprintf("Unkown module '%s'", moduleName), 400)
		redisRequestErrors.Inc()
		return
	}
	log.Debugf("Scraping target '%s' with module '%s'", target, moduleName)

	start := time.Now()
	registry := prometheus.NewRegistry()
//	collector := collector{target: target, module: module}
//	registry.MustRegister(collector)

	// Delegate http serving to Promethues client library, which will call collector.Collect.
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	duration := float64(time.Since(start).Seconds())
	redisDuration.WithLabelValues(moduleName).Observe(duration)
	log.Debugf("Scrape of target '%s' with module '%s' took %f seconds", target, moduleName, duration)
}



func main() {
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
            <label>Module:</label> <input type="text" name="module" placeholder="module" value="default"><br>
            <input type="submit" value="Submit">
            </form>
            <p><a href="/metrics">Metrics</a></p>
            </body>
            </html>`))
	})

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/redis", handler)
	log.Fatal(http.ListenAndServe(":9200", nil))
}
