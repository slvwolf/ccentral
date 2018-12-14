package plugins

import (
	"log"
	"reflect"
	"regexp"
	"strings"
)

var valueRe = regexp.MustCompile(`[^a-zA-Z0-9_:]`)

// MetricPrefixHistogram - Prefix for histrogram data
const MetricPrefixHistogram = "h_"

// MetricPrefixCounter - Prefix for counter data
const MetricPrefixCounter = "c_"

// UnixTime - Provides unix epoch in seconds
type UnixTime interface {
	Unix() int64
}

// CleanValue - Converts value to a valid CCentral name
func CleanValue(value string) string {
	return valueRe.ReplaceAllString(strings.ToLower(value), ``)
}

// HistogramPoint -
type HistogramPoint struct {
	Key           string
	Percentile75  int
	Percentile95  int
	Percentile99  int
	PercentileMed int
}

func newHistogramPoint(key string, percentiles []interface{}) (*HistogramPoint, bool) {
	if len(percentiles) != 4 {
		return nil, false
	}
	p75, found75 := percentiles[0].(float64)
	p95, found95 := percentiles[1].(float64)
	p99, found99 := percentiles[2].(float64)
	med, foundMed := percentiles[3].(float64)
	if !found75 || !found95 || !found99 || !foundMed {
		return nil, false
	}
	m := &HistogramPoint{Key: key, Percentile75: int(p75), Percentile95: int(p95), Percentile99: int(p99), PercentileMed: int(med)}
	return m, true
}

// Add - Adds up to histograms (averages the result)
func (p *HistogramPoint) Add(i *HistogramPoint) {
	p.Percentile75 = (i.Percentile75 + p.Percentile75) / 2
	p.Percentile95 = (i.Percentile95 + p.Percentile95) / 2
	p.Percentile99 = (i.Percentile99 + p.Percentile99) / 2
	p.PercentileMed = (i.PercentileMed + p.PercentileMed) / 2
}

// CollectHistograms - Collect all histograms and calculate a single histrogram from all of the instances
func CollectHistograms(data map[string]interface{}, histograms map[string]*HistogramPoint) map[string]*HistogramPoint {
	for key, value := range data {
		if strings.HasPrefix(key, MetricPrefixHistogram) {
			cList, found := value.([]interface{})
			if !found {
				log.Printf("Problem collecting histograms, expected a list but got: %T", value)
				continue
			}
			if newGram, ok := newHistogramPoint(key, cList); ok {
				if oldGram, ok := histograms[key]; ok {
					oldGram.Add(newGram)
				} else {
					histograms[key] = newGram
				}
			} else {
				log.Printf("Problem collecting histograms, could not map value")
				continue
			}
		}
	}
	return histograms
}

// CollectInstanceCounters - Collects all instance counter data
func CollectInstanceCounters(data map[string]interface{}, counters map[string]int) map[string]int {
	for key, value := range data {
		if strings.HasPrefix(key, MetricPrefixCounter) {
			cList, found := value.([]interface{})
			if !found {
				log.Printf("Problem collecting counters, expected a list but got: %T", value)
				continue
			}
			if len(cList) < 1 {
				continue
			}
			v := cList[len(cList)-1]
			iValue, found := v.(float64)
			if !found {
				log.Printf("Problem collecting counters, list contained unsupported type: " + reflect.TypeOf(v).Name())
				continue
			}
			if val, ok := counters[key]; ok {
				counters[key] = val + int(iValue)
			} else {
				counters[key] = int(iValue)
			}
		}
	}
	return counters
}
