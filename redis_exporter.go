package main

import (
	"fmt"
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

	"github.com/shakapark/Redis-Exporter/config"
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

	passwd = kingpin.Flag("password","Password to Redis Database").Default("").String()

	sc = &config.SafeConfig{C: &config.Config{},}

	configFile = kingpin.Flag("config.file", "Redis exporter configuration file.").Default("redis.yml").String()
	listenAddress = kingpin.Flag("web.listen-address", "The address to listen on for HTTP requests.").Default(":9140").String()
)

func init() {
	prometheus.MustRegister(redisDuration)
	prometheus.MustRegister(redisRequestErrors)
	prometheus.MustRegister(version.NewCollector("redis_exporter"))
}

func handler(w http.ResponseWriter, r *http.Request, c *config.Config) {

	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "'target' parameter must be specified", 400)
		redisRequestErrors.Inc()
		return
	}

	objectName := r.URL.Query().Get("object")
	if objectName == "" {
		http.Error(w, "'object' parameter must be specified", 400)
		redisRequestErrors.Inc()
		return
	}

	object, ok := c.Objects[objectName]
	if !ok {
		http.Error(w, fmt.Sprintf("Unknown object %q", objectName), 400)
		return
	}

	log.Debugf("Scraping target '%s' with object '%s'", target, objectName)

	start := time.Now()
	registry := prometheus.NewRegistry()
	collector := collector{target: target, object: object, passwd: *passwd}
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

	if err := sc.ReloadConfig(*configFile); err != nil {
		log.Fatalf("Error loading config: %s", err)
	}
	log.Infoln("Loaded config file")

	log.Infoln("Starting redis exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	hup := make(chan os.Signal)
	reloadCh := make(chan chan error)
	signal.Notify(hup, syscall.SIGHUP)

	go func() {
		for {
			select {
			case <-hup:
				if err := sc.ReloadConfig(*configFile); err != nil {
					log.Errorf("Error reloading config: %s", err)
					continue
				}
				log.Infoln("Loaded config file")
			case rc := <-reloadCh:
				if err := sc.ReloadConfig(*configFile); err != nil {
					log.Errorf("Error reloading config: %s", err)
					rc <- err
				} else {
					log.Infoln("Loaded config file")
					rc <- nil
				}
			}
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
		<html>
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
				<br><br><br>
				<form action="/-/reload" method="POST">
					<input type="submit" value="reload">
				</form>
			</body>
		</html>`))
	})

	http.HandleFunc("/-/reload",
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				fmt.Fprintf(w, "This endpoint requires a POST request.\n")
				return
			}

			rc := make(chan error)
			reloadCh <- rc
			if err := <-rc; err != nil {
				http.Error(w, fmt.Sprintf("failed to reload config: %s", err), http.StatusInternalServerError)
			}
		})

	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/redis", func(w http.ResponseWriter, r *http.Request) {
		sc.Lock()
		conf := sc.C
		sc.Unlock()
		handler(w, r, conf)
	})
	log.Infof("Listening on ", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		log.Fatalf("Error starting HTTP server: %s", err)
	}
}
