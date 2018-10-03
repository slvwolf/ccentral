/*
Prometheus exporter.
* https://prometheus.io/docs/instrumenting/exposition_formats/

Exposes all counters and numerical values as gauges. Formatting follows,

# TYPE cc_[service]_[metric] gauge
cc_[service]_[metric] VALUE TS
*/
package plugins

import (
	"bytes"
	"fmt"
	"log"

	"github.com/slvwolf/ccentral/client"
)

// UnixTime - Provides unix epoch in seconds
type UnixTime interface {
	Unix() int64
}

// GeneratePrometheusPayload - Collect and create Prometheus payload
func GeneratePrometheusPayload(cc client.CCServerReadApi, unixTime UnixTime) ([]byte, error) {
	epoch := unixTime.Unix()
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

		// Write active instance count
		buffer.WriteString(fmt.Sprintf("# TYPE cc_%s_instances gauge\n", serviceID))
		buffer.WriteString(fmt.Sprintf("cc_%s_instances %d %d\n", serviceID, count, epoch))

		for _, instance := range instances {
			log.Printf("Collecting counters for %v", serviceID)
			counters = collectInstanceCounters(instance, counters)
		}
		for key, value := range counters {
			buffer.WriteString(fmt.Sprintf("# TYPE cc_%s_%s gauge\n", serviceID, key))
			buffer.WriteString(fmt.Sprintf("cc_%s_%s %d %d\n", serviceID, key, value, epoch))
		}
	}
	return buffer.Bytes(), nil
}
