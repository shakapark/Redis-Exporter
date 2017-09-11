package main

import(
	//"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"

	"github.com/shakapark/Redis-Exporter/config"
)

var (

)

func StringConvert(a []uint8) string{
	b := make([]byte, 0, len(a))
	for _, i := range a {
		b = append(b, byte(i))
	}
	return string(b)
}

func NombreConvert(a []uint8) float64{
	return float64(a)
}

type collector struct {
	target string
	object config.Object
}

func (c collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("dummy", "dummy", nil, nil)
}

func (c collector) Collect(ch chan<- prometheus.Metric){
	start := time.Now()

	conn, err := redis.Dial("tcp", c.target)
	if err != nil {
		log.Infof("Error scraping target %s: %s", c.target, err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("redis_error", "Error scraping target", nil, nil), err)
		return
	}

	r, err := conn.Do("GET",c.object.Name)
	if err != nil {
		log.Infof("Result : %s", err)
	}

	//Suite à modifier suivant object.Type
	switch c.object.Type {
		case "nombre":
			result := NombreConvert(r.([]uint8))
		case "text":
			result := StringConvert(r.([]uint8))
		default:
	}

	defer conn.Close()

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("redis_scrape_duration_seconds", "Total REDIS time scrape took.", nil, nil),
		prometheus.GaugeValue,
		float64(time.Since(start).Seconds()))
}

