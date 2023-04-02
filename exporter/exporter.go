package exporter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
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

var (
	strToMetricsType map[string]metricsType

	ConflictErr = errors.New("metrics conflict")
)

type metricsRecipe struct {
	Spec spec          `yaml:"spec"`
	Data []metricsData `yaml:"data"`
}

type spec struct {
	Name   string   `yaml:"name"`
	Type   string   `yaml:"type"`
	Labels []string `yaml:"labels"`
}

type metricsData struct {
	Labels   []label `yaml:"labels"`
	Sequence string  `yaml:"sequence"`
}

type label struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type parsedMetricsData struct {
	labels   map[string]string
	sequence []int
}

type counterExporter struct {
	counterVec        *prometheus.CounterVec
	parsedMetricsData []*parsedMetricsData
}

type gaugeExporter struct {
	gaugeVec          *prometheus.GaugeVec
	parsedMetricsData []*parsedMetricsData
}

var types map[string]metricsType

var counterExporters map[string]*counterExporter
var gaugeExporters map[string]*gaugeExporter

var mu sync.Mutex

func init() {
	strToMetricsType = make(map[string]metricsType)
	strToMetricsType["counter"] = Counter
	strToMetricsType["gauge"] = Gauge

	types = make(map[string]metricsType)

	counterExporters = make(map[string]*counterExporter)
	gaugeExporters = make(map[string]*gaugeExporter)
}

func parseSequence(sequence string) ([]int, error) {
	result := make([]int, 0)

	tokens := strings.Split(sequence, " ")
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
					return nil, fmt.Errorf("invalid values format %s", sequence)
				}
			} else {
				return nil, fmt.Errorf("invalid values format %s", sequence)
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

func unmarshalAllRecipe(in []byte, out *[]metricsRecipe) error {
	r := bytes.NewReader(in)
	decoder := yaml.NewDecoder(r)
	for {
		var mr metricsRecipe
		if err := decoder.Decode(&mr); err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		*out = append(*out, mr)
	}
	return nil
}

func conflict(recipe []metricsRecipe) (bool, int) {
	for i, r := range recipe {
		if _, ok := counterExporters[r.Spec.Name]; ok {
			return true, i
		}
		if _, ok := gaugeExporters[r.Spec.Name]; ok {
			return true, i
		}
	}
	return false, -1
}

func invalidSpec(recipe []metricsRecipe) (bool, int) {
	for i, r := range recipe {
		if r.Spec.Name == "" {
			return true, i
		}
		if _, ok := strToMetricsType[r.Spec.Type]; !ok {
			return true, i
		}
		if len(r.Spec.Labels) == 0 {
			return true, i
		}
	}
	return false, -1
}

func invalidDataLabel(specLabel []string, dataLabel map[string]string) bool {
	if len(specLabel) != len(dataLabel) {
		return true
	}

	for _, sl := range specLabel {
		if _, ok := dataLabel[sl]; !ok {
			return true
		}
	}
	return false
}

func Register(yamlData []byte) error {
	mu.Lock()
	defer mu.Unlock()

	var recipe []metricsRecipe
	err := unmarshalAllRecipe(yamlData, &recipe)
	if err != nil {
		return err
	}

	if result, i := conflict(recipe); result {
		return fmt.Errorf("%s: %w", recipe[i].Spec.Name, ConflictErr)
	}

	if result, i := invalidSpec(recipe); result {
		return fmt.Errorf("invalid metrics spec. name: %s, type: %s, labels: %v",
			recipe[i].Spec.Name, recipe[i].Spec.Type, recipe[i].Spec.Labels)
	}

	for _, r := range recipe {
		switch strToMetricsType[r.Spec.Type] {
		case Counter:
			if _, ok := counterExporters[r.Spec.Name]; ok {
				clearSpecifiedMetrics(r.Spec.Name)
			}

			counterVec := promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: r.Spec.Name,
				},
				r.Spec.Labels,
			)

			var pmds []*parsedMetricsData
			for _, metData := range r.Data {
				parsedSeq, err := parseSequence(metData.Sequence)
				if err != nil {
					return err
				}

				labels := make(map[string]string)
				for _, l := range metData.Labels {
					labels[l.Key] = l.Value
				}
				if invalidDataLabel(r.Spec.Labels, labels) {
					return fmt.Errorf("data label is invalid: %v", labels)
				}

				pmds = append(pmds, &parsedMetricsData{
					labels:   labels,
					sequence: parsedSeq,
				})
			}
			counterExporters[r.Spec.Name] = &counterExporter{
				counterVec:        counterVec,
				parsedMetricsData: pmds,
			}
		case Gauge:
			if _, ok := gaugeExporters[r.Spec.Name]; ok {
				clearSpecifiedMetrics(r.Spec.Name)
			}

			gaugeVec := promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: r.Spec.Name,
				},
				r.Spec.Labels,
			)

			var pmds []*parsedMetricsData
			for _, metData := range r.Data {
				parsedSeq, err := parseSequence(metData.Sequence)
				if err != nil {
					return err
				}

				labels := make(map[string]string)
				for _, l := range metData.Labels {
					labels[l.Key] = l.Value
				}
				if invalidDataLabel(r.Spec.Labels, labels) {
					return fmt.Errorf("data label is invalid: %v", labels)
				}

				pmds = append(pmds, &parsedMetricsData{
					labels:   labels,
					sequence: parsedSeq,
				})
			}
			gaugeExporters[r.Spec.Name] = &gaugeExporter{
				gaugeVec:          gaugeVec,
				parsedMetricsData: pmds,
			}
		default:
			panic(fmt.Sprintf("unknown type: %d", types[r.Spec.Type]))
		}
		types[r.Spec.Name] = strToMetricsType[r.Spec.Type]
	}

	return nil
}

func deleteEntriesFromParsedMetricsData(toBeDeletedDataIndex []int, parsedMetricsData []*parsedMetricsData) []*parsedMetricsData {
	sort.Slice(toBeDeletedDataIndex, func(i, j int) bool {
		return toBeDeletedDataIndex[i] > toBeDeletedDataIndex[j]
	})
	for _, delIndex := range toBeDeletedDataIndex {
		parsedMetricsData[delIndex] = parsedMetricsData[len(parsedMetricsData)-1]
		parsedMetricsData = parsedMetricsData[:len(parsedMetricsData)-1]
	}
	return parsedMetricsData
}

func Update() {
	mu.Lock()
	defer mu.Unlock()

	for metName, exporter := range counterExporters {
		if len(exporter.parsedMetricsData) == 0 {
			continue
		}
		toBeDeletedDataIndex := make([]int, 0)
		for i, pmd := range exporter.parsedMetricsData {
			exporter.counterVec.With(pmd.labels).Add(float64(pmd.sequence[0]))
			pmd.sequence = pmd.sequence[1:]
			if len(pmd.sequence) == 0 {
				log.Printf("empty value found for %s.", metName)
				toBeDeletedDataIndex = append(toBeDeletedDataIndex, i)
			}
		}
		exporter.parsedMetricsData = deleteEntriesFromParsedMetricsData(toBeDeletedDataIndex, exporter.parsedMetricsData)
	}

	for metName, exporter := range gaugeExporters {
		if len(exporter.parsedMetricsData) == 0 {
			continue
		}
		toBeDeletedDataIndex := make([]int, 0)
		for i, pmd := range exporter.parsedMetricsData {
			exporter.gaugeVec.With(pmd.labels).Set(float64(pmd.sequence[0]))
			pmd.sequence = pmd.sequence[1:]
			if len(pmd.sequence) == 0 {
				log.Printf("empty value found for %s.", metName)
				toBeDeletedDataIndex = append(toBeDeletedDataIndex, i)
			}
		}
		exporter.parsedMetricsData = deleteEntriesFromParsedMetricsData(toBeDeletedDataIndex, exporter.parsedMetricsData)
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
				if len(pmd.sequence) != 0 {
					shouldBeRemoved = false
				}
			}
		}
		if shouldBeRemoved {
			toBeDeletedMetrics = append(toBeDeletedMetrics, metName)
		}
	}

	for metName, exporter := range gaugeExporters {
		shouldBeRemoved := true
		if !force {
			for _, pmd := range exporter.parsedMetricsData {
				if len(pmd.sequence) != 0 {
					shouldBeRemoved = false
				}
			}
		}
		if shouldBeRemoved {
			toBeDeletedMetrics = append(toBeDeletedMetrics, metName)
		}
	}
	for _, metName := range toBeDeletedMetrics {
		clearSpecifiedMetrics(metName)
	}
}

// Lock should be acquired by the caller.
func clearSpecifiedMetrics(metricsName string) {
	switch types[metricsName] {
	case Counter:
		if !prometheus.Unregister(counterExporters[metricsName].counterVec) {
			log.Printf("unregister failed. metricsName = %s", metricsName)
		}
		delete(counterExporters, metricsName)
	case Gauge:
		if !prometheus.Unregister(gaugeExporters[metricsName].gaugeVec) {
			log.Printf("unregister failed. metricsName = %s", metricsName)
		}
		delete(gaugeExporters, metricsName)
	}
	delete(types, metricsName)

	log.Printf("metrics %v was removed", metricsName)
}
