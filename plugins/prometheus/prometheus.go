/*
Prometheus exporter.
* https://prometheus.io/docs/instrumenting/exposition_formats/

Exposes all counters and numerical values as gauges. Formatting follows,

# TYPE cc_[service]_[metric] gauge
cc_[service]_[metric] VALUE TS
*/
package prometheus

import (
	"bytes"
	"fmt"
	"log"

	"github.com/slvwolf/ccentral/client"
	"github.com/slvwolf/ccentral/plugins"
)

func toMetricText(p *plugins.HistogramPoint, cleanServiceID string, epoch int64) []byte {
	var buffer bytes.Buffer
	cleanKey := plugins.CleanValue(p.Key)
	buffer.WriteString(fmt.Sprintf("# TYPE cc_%s_%s gauge\n", cleanServiceID, cleanKey))
	buffer.WriteString(fmt.Sprintf("cc_%s_%s{percentile=\"75\"} %d %d\n", cleanServiceID, cleanKey, p.Percentile75, epoch))
	buffer.WriteString(fmt.Sprintf("cc_%s_%s{percentile=\"95\"} %d %d\n", cleanServiceID, cleanKey, p.Percentile95, epoch))
	buffer.WriteString(fmt.Sprintf("cc_%s_%s{percentile=\"99\"} %d %d\n", cleanServiceID, cleanKey, p.Percentile99, epoch))
	buffer.WriteString(fmt.Sprintf("cc_%s_%s{percentile=\"median\"} %d %d\n", cleanServiceID, cleanKey, p.PercentileMed, epoch))
	return buffer.Bytes()
}

// GeneratePrometheusPayload - Collect and create Prometheus payload
func GeneratePrometheusPayload(cc client.CCServerReadApi, unixTime plugins.UnixTime) ([]byte, error) {
	epoch := unixTime.Unix() * 1000
	var buffer bytes.Buffer
	serviceList, err := cc.GetServiceList()
	if err != nil {
		log.Printf("WARN Could not retrieve service list")
		return nil, err
	}
	for _, serviceID := range serviceList.Services {
		log.Printf("Handling service %v", serviceID)
		instances, err := cc.GetInstanceList(serviceID)
		if err != nil {
			log.Printf("WARN Could not retrieve instance list")
			return nil, err
		}
		count := len(instances)
		counters := make(map[string]int)
		histograms := make(map[string]*plugins.HistogramPoint)

		cleanServiceID := plugins.CleanValue(serviceID)

		// Write active instance count
		buffer.WriteString(fmt.Sprintf("# TYPE cc_%s_instances gauge\n", cleanServiceID))
		buffer.WriteString(fmt.Sprintf("cc_%s_instances %d %d\n", cleanServiceID, count, epoch))

		for _, instance := range instances {
			counters = plugins.CollectInstanceCounters(instance, counters)
			histograms = plugins.CollectHistograms(instance, histograms)
			log.Printf("Total %d counters and %d histograms for %v", len(counters), len(histograms), serviceID)
		}
		for key, value := range counters {
			cleanKey := plugins.CleanValue(key)
			buffer.WriteString(fmt.Sprintf("# TYPE cc_%s_%s gauge\n", cleanServiceID, cleanKey))
			buffer.WriteString(fmt.Sprintf("cc_%s_%s %d %d\n", cleanServiceID, cleanKey, value, epoch))
		}
		for _, value := range histograms {
			buffer.Write(toMetricText(value, cleanServiceID, epoch))
		}
	}
	return buffer.Bytes(), nil
}
