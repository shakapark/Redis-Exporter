package main

import(
	//"fmt"
	"time"
	
	"github.com/garyburd/redigo/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

var (

)

func Convert(a []uint8) string{
	b := make([]byte, 0, len(a))
	for _, i := range a {
		b = append(b, byte(i))
	}
	return string(b)
}

type collector struct {
	target string
	object string
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
	
	r, err := conn.Do("GET",c.object)
	if err != nil {
		log.Infof("Result : %s", err)
	}
	result := Convert(r.([]uint8))
	log.Infof("Result : %v", result)
		
	defer conn.Close()
	
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("redis_scrape_duration_seconds", "Total REDIS time scrape took.", nil, nil),
		prometheus.GaugeValue,
		float64(time.Since(start).Seconds()))
}

