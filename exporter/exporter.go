package exporter

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gopkg.in/yaml.v2"
)

type metricsType int

const (
	Counter metricsType = iota
	Gauge
	Histogram
)

var strToMetricsType map[string]metricsType

type metricsRecipe struct {
	Metrics     []metrics     `yaml:"metrics"`
	InputSeries []inputSeries `yaml:"input_series"`
}

type metrics struct {
	Name   string   `yaml:"name"`
	Type   string   `yaml:"type"`
	Labels []string `yaml:"labels"`
}

type inputSeries struct {
	MetricsName string        `yaml:"metrics_name"`
	Data        []metricsData `yaml:"data"`
}

type metricsData struct {
	Labels []label `yaml:"labels"`
	Values string  `yaml:"values"`
}

type label struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type parsedMetricsData struct {
	labels map[string]string
	values []int
}

type counterExporter struct {
	metricsName       string
	counterVec        *prometheus.CounterVec
	parsedMetricsData []*parsedMetricsData
}

type gaugeExporter struct {
	metricsName       string
	gaugeVec          *prometheus.GaugeVec
	parsedMetricsData []*parsedMetricsData
}

var counters map[string]*prometheus.CounterVec
var gauges map[string]*prometheus.GaugeVec
var types map[string]metricsType

var counterExporters map[string]*counterExporter
var gaugeExporters map[string]*gaugeExporter

var mu sync.Mutex

func init() {
	strToMetricsType = make(map[string]metricsType)
	strToMetricsType["counter"] = Counter
	strToMetricsType["gauge"] = Gauge

	counters = make(map[string]*prometheus.CounterVec)
	gauges = make(map[string]*prometheus.GaugeVec)
	types = make(map[string]metricsType)

	counterExporters = make(map[string]*counterExporter)
	gaugeExporters = make(map[string]*gaugeExporter)
}

func parseValues(values string) ([]int, error) {
	result := make([]int, 0)

	tokens := strings.Split(values, " ")
	for _, token := range tokens {
		if strings.Contains(token, "x") {
			initStr := ""
			stepStr := ""
			timesStr := ""
			tmpToken := strings.Split(token, "+")
			if len(tmpToken) == 2 {
				// 1+2x3 or -1+2x3 style
				initStr = tmpToken[0]
				stepAndTimes := strings.Split(tmpToken[1], "x")
				stepStr = stepAndTimes[0]
				timesStr = stepAndTimes[1]
			} else if len(tmpToken) == 1 {
				tmpToken = strings.Split(token, "-")
				if len(tmpToken) == 2 {
					// 1-2x3 style
					initStr = tmpToken[0]
					stepAndTimes := strings.Split(tmpToken[1], "x")
					stepStr = "-" + stepAndTimes[0]
					timesStr = stepAndTimes[1]
				} else if len(tmpToken) == 3 {
					// -1-2x3
					initStr = "-" + tmpToken[1]
					stepAndTimes := strings.Split(tmpToken[2], "x")
					stepStr = "-" + stepAndTimes[0]
					timesStr = stepAndTimes[1]
				} else if len(tmpToken) == 1 {
					// 1x3 style (shorthand for '1+0x3')
					initAndTimes := strings.Split(tmpToken[0], "x")
					initStr = initAndTimes[0]
					timesStr = initAndTimes[1]
					stepStr = "0"
				} else {
					return nil, fmt.Errorf("invalid values format %s", values)
				}
			} else {
				return nil, fmt.Errorf("invalid values format %s", values)
			}

			init, err := strconv.Atoi(initStr)
			if err != nil {
				return nil, err
			}
			step, err := strconv.Atoi(stepStr)
			if err != nil {
				return nil, err
			}
			times, err := strconv.Atoi(timesStr)
			if err != nil {
				return nil, err
			}

			result = append(result, init)
			for i := 0; i < times; i++ {
				lastVal := result[len(result)-1]
				result = append(result, lastVal+step)
			}
		} else {
			// Just a single number
			val, err := strconv.Atoi(token)
			if err != nil {
				return nil, err
			}
			result = append(result, val)
		}
	}

	return result, nil
}

func Register(yamlData []byte) error {
	mu.Lock()
	defer mu.Unlock()

	var recipe metricsRecipe
	err := yaml.Unmarshal(yamlData, &recipe)
	if err != nil {
		return err
	}

	for _, m := range recipe.Metrics {
		var ok bool
		types[m.Name], ok = strToMetricsType[m.Type]
		if !ok {
			return fmt.Errorf("invalid metrics type %s specified for %s", m.Type, m.Name)
		}
		switch types[m.Name] {
		case Counter:
			counters[m.Name] = promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: m.Name,
				},
				m.Labels,
			)
		case Gauge:
			gauges[m.Name] = promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: m.Name,
				},
				m.Labels,
			)
		default:
			panic(fmt.Sprintf("unknown type: %d", types[m.Name]))
		}
	}

	for _, is := range recipe.InputSeries {
		for _, metData := range is.Data {
			parsedValues, err := parseValues(metData.Values)
			if err != nil {
				return err
			}

			labels := make(map[string]string)
			for _, l := range metData.Labels {
				labels[l.Key] = l.Value
			}
			pmd := &parsedMetricsData{
				labels: labels,
				values: parsedValues,
			}
			switch types[is.MetricsName] {
			case Counter:
				if _, ok := counterExporters[is.MetricsName]; !ok {
					if _, ok := counters[is.MetricsName]; !ok {
						return fmt.Errorf("metrics definition not found: %s", is.MetricsName)
					}
					counterExporters[is.MetricsName] = &counterExporter{
						metricsName: is.MetricsName,
						counterVec:  counters[is.MetricsName],
						parsedMetricsData: []*parsedMetricsData{
							pmd,
						},
					}
				} else {
					counterExporters[is.MetricsName].parsedMetricsData = append(counterExporters[is.MetricsName].parsedMetricsData, pmd)
				}
			case Gauge:
				if _, ok := gaugeExporters[is.MetricsName]; !ok {
					if _, ok := gauges[is.MetricsName]; !ok {
						return fmt.Errorf("metrics definition not found: %s", is.MetricsName)
					}
					gaugeExporters[is.MetricsName] = &gaugeExporter{
						metricsName: is.MetricsName,
						gaugeVec:    gauges[is.MetricsName],
						parsedMetricsData: []*parsedMetricsData{
							pmd,
						},
					}
				} else {
					gaugeExporters[is.MetricsName].parsedMetricsData = append(gaugeExporters[is.MetricsName].parsedMetricsData, pmd)
				}
			}
		}
	}

	return nil
}

func Update() {
	mu.Lock()
	defer mu.Unlock()

	for _, exporter := range counterExporters {
		for _, pmd := range exporter.parsedMetricsData {
			if len(pmd.values) == 0 {
				log.Printf("empty value found for %s.", exporter.metricsName)
				continue
			}
			exporter.counterVec.With(pmd.labels).Add(float64(pmd.values[0]))
			pmd.values = pmd.values[1:]
		}
	}

	for _, exporter := range gaugeExporters {
		for _, pmd := range exporter.parsedMetricsData {
			if len(pmd.values) == 0 {
				log.Printf("empty value found for %s.", exporter.metricsName)
				continue
			}
			exporter.gaugeVec.With(pmd.labels).Set(float64(pmd.values[0]))
			pmd.values = pmd.values[1:]
		}
	}
}

func Clear(force bool) {
	mu.Lock()
	defer mu.Unlock()

	toBeDeletedMetrics := make([]string, 0)
	for metName, exporter := range counterExporters {
		shouldBeRemoved := true
		if !force {
			for _, pmd := range exporter.parsedMetricsData {
				if len(pmd.values) != 0 {
					shouldBeRemoved = false
				}
			}
		}
		if shouldBeRemoved {
			log.Printf("metrics %v is to be removed", metName)
			if !prometheus.Unregister(exporter.counterVec) {
				log.Printf("unregister failed. metricsName = %s", metName)
			}
			toBeDeletedMetrics = append(toBeDeletedMetrics, metName)
		}
	}
	for _, metName := range toBeDeletedMetrics {
		delete(counterExporters, metName)
		delete(counters, metName)
		delete(types, metName)
	}

	toBeDeletedMetrics = make([]string, 0)
	for metName, exporter := range gaugeExporters {
		shouldBeRemoved := true
		if !force {
			for _, pmd := range exporter.parsedMetricsData {
				if len(pmd.values) != 0 {
					shouldBeRemoved = false
				}
			}
		}
		if shouldBeRemoved {
			log.Printf("metrics %v is to be removed", metName)
			if !prometheus.Unregister(exporter.gaugeVec) {
				log.Printf("unregister failed. metricsName = %s", metName)
			}
			toBeDeletedMetrics = append(toBeDeletedMetrics, metName)
		}
	}
	for _, metName := range toBeDeletedMetrics {
		delete(gaugeExporters, metName)
		delete(gauges, metName)
		delete(types, metName)
	}
}
