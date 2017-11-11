package main

import(
	//"encoding/json"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"

	"github.com/shakapark/Redis-Exporter/config"
)

var (

)

func StringConverter(a []uint8) string{
	b := make([]byte, 0, len(a))
	for _, i := range a {
		b = append(b, byte(i))
	}
	return string(b)
}

/*
// Function to copy for more json Type and change x value
func jsonxConverter(s string, ch chan<- prometheus.Metric, c collector) {

	type Json struct {
		// Structur : get help to https://mholt.github.io/json-to-go/ Try to not have interface{} type
	}
	
	var data Json
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		log.Infof("err", err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("redis_error", "Error scraping target", nil, nil), err)
		return
	}
	
	// Now put all values you want to export in the tab
	tab := make(map[string]float64)
	
	tab["label1"] = data.label1
	tab["label2"] = float64(data.label.label2) // Use float64(...) If value has not good type
	[...]
	
	for k, v := range tab {
	
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("redis_get_"+c.object.Name, "Result to Redis GET.", []string{"label"}, nil),
			prometheus.GaugeValue,
			v, k)	
	}

}
// Function to copy for more json Type and change x value
*/
type collector struct {
	target string
	object config.Object
	passwd string
}

func (c collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("dummy", "dummy", nil, nil)
}

func (c collector) Collect(ch chan<- prometheus.Metric){
	start := time.Now()

	conn, err := redis.Dial("tcp", c.target, redis.DialPassword(c.passwd))
	if err != nil {
		log.Infof("Error scraping target %s: %s", c.target, err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("redis_error", "Error scraping target", nil, nil), err)
		return
	}

	r, err := conn.Do("GET",c.object.Name)
	if err != nil {
		log.Infof("Result : %s", err)
	}

	switch c.object.Type {
		case "nombre":
			rTemp := StringConverter(r.([]uint8))
			result , err := strconv.ParseFloat(rTemp, 64)
			if err == nil {
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("redis_get_"+c.object.Name, "Redis Get result", nil, nil),
					prometheus.GaugeValue,
					result)
			}else{
				ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("redis_error", "Error Convert Object", nil, nil), err)
			}
		case "text":
			result := StringConverter(r.([]uint8))
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("redis_get_"+c.object.Name, "Total REDIS time scrape took.", []string{"result"}, nil),
				prometheus.GaugeValue,
				1,
				result)
		
		// Modul to copy for more json Type and change x value
//		case "jsonx":
//			jsonxConverter(StringConverter(r.([]uint8)), ch, c)
		// Modul to copy for more json Type and change x value
			
		default:
			ch <- prometheus.NewInvalidMetric(prometheus.NewDesc("redis_error", "Unknown Type", nil, nil), nil)
	}

	defer conn.Close()

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("redis_scrape_duration_seconds", "Total REDIS time scrape took.", nil, nil),
		prometheus.GaugeValue,
		float64(time.Since(start).Seconds()))
}

