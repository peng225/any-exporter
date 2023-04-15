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
	Name    string    `yaml:"name"`
	Type    string    `yaml:"type"`
	Labels  []string  `yaml:"labels"`
	Buckets []float64 `yaml:"buckets"`
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
	sequence []float64
	prevData float64
}

type counterExporter struct {
	counterVec        *prometheus.CounterVec
	parsedMetricsData []*parsedMetricsData
}

func newCounterExporter(recipe *metricsRecipe) (*counterExporter, error) {
	counterVec := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: recipe.Spec.Name,
		},
		recipe.Spec.Labels,
	)

	var pmds []*parsedMetricsData
	for _, metData := range recipe.Data {
		parsedSeq, err := parseSequence(metData.Sequence)
		if err != nil {
			return nil, err
		}

		if !ascending(parsedSeq) {
			return nil, fmt.Errorf("sequence must be in the ascending order.")
		}

		labels := make(map[string]string)
		for _, l := range metData.Labels {
			labels[l.Key] = l.Value
		}
		if invalidDataLabel(recipe.Spec.Labels, labels) {
			return nil, fmt.Errorf("data label is invalid: %v", labels)
		}

		pmds = append(pmds, &parsedMetricsData{
			labels:   labels,
			sequence: parsedSeq,
		})
	}

	return &counterExporter{
		counterVec:        counterVec,
		parsedMetricsData: pmds,
	}, nil
}

func (ce *counterExporter) update(metName string) {
	if len(ce.parsedMetricsData) == 0 {
		return
	}
	toBeDeletedDataIndex := make([]int, 0)
	for i, pmd := range ce.parsedMetricsData {
		ce.counterVec.With(pmd.labels).Add(float64(pmd.sequence[0]) - pmd.prevData)
		pmd.prevData = pmd.sequence[0]
		pmd.sequence = pmd.sequence[1:]
		if len(pmd.sequence) == 0 {
			log.Printf("empty value found for %s.", metName)
			toBeDeletedDataIndex = append(toBeDeletedDataIndex, i)
		}
	}
	ce.parsedMetricsData = deleteEntriesFromParsedMetricsData(toBeDeletedDataIndex, ce.parsedMetricsData)
}

type gaugeExporter struct {
	gaugeVec          *prometheus.GaugeVec
	parsedMetricsData []*parsedMetricsData
}

func newGaugeExporter(recipe *metricsRecipe) (*gaugeExporter, error) {
	gaugeVec := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: recipe.Spec.Name,
		},
		recipe.Spec.Labels,
	)

	var pmds []*parsedMetricsData
	for _, metData := range recipe.Data {
		parsedSeq, err := parseSequence(metData.Sequence)
		if err != nil {
			return nil, err
		}

		labels := make(map[string]string)
		for _, l := range metData.Labels {
			labels[l.Key] = l.Value
		}
		if invalidDataLabel(recipe.Spec.Labels, labels) {
			return nil, fmt.Errorf("data label is invalid: %v", labels)
		}

		pmds = append(pmds, &parsedMetricsData{
			labels:   labels,
			sequence: parsedSeq,
		})
	}

	return &gaugeExporter{
		gaugeVec:          gaugeVec,
		parsedMetricsData: pmds,
	}, nil
}

func (ga *gaugeExporter) update(metName string) {
	if len(ga.parsedMetricsData) == 0 {
		return
	}
	toBeDeletedDataIndex := make([]int, 0)
	for i, pmd := range ga.parsedMetricsData {
		ga.gaugeVec.With(pmd.labels).Set(float64(pmd.sequence[0]))
		pmd.sequence = pmd.sequence[1:]
		if len(pmd.sequence) == 0 {
			log.Printf("empty value found for %s.", metName)
			toBeDeletedDataIndex = append(toBeDeletedDataIndex, i)
		}
	}
	ga.parsedMetricsData = deleteEntriesFromParsedMetricsData(toBeDeletedDataIndex, ga.parsedMetricsData)
}

type histogramExporter struct {
	histogramVec      *prometheus.HistogramVec
	parsedMetricsData []*parsedMetricsData
}

func newHistogramExporter(recipe *metricsRecipe) (*histogramExporter, error) {
	histogramVec := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    recipe.Spec.Name,
			Buckets: recipe.Spec.Buckets,
		},
		recipe.Spec.Labels,
	)

	var pmds []*parsedMetricsData
	for _, metData := range recipe.Data {
		parsedSeq, err := parseSequence(metData.Sequence)
		if err != nil {
			return nil, err
		}

		labels := make(map[string]string)
		for _, l := range metData.Labels {
			labels[l.Key] = l.Value
		}
		if invalidDataLabel(recipe.Spec.Labels, labels) {
			return nil, fmt.Errorf("data label is invalid: %v", labels)
		}

		pmds = append(pmds, &parsedMetricsData{
			labels:   labels,
			sequence: parsedSeq,
		})
	}

	return &histogramExporter{
		histogramVec:      histogramVec,
		parsedMetricsData: pmds,
	}, nil
}

func (hi *histogramExporter) update(metName string) {
	if len(hi.parsedMetricsData) == 0 {
		return
	}
	toBeDeletedDataIndex := make([]int, 0)
	for i, pmd := range hi.parsedMetricsData {
		hi.histogramVec.With(pmd.labels).Observe(float64(pmd.sequence[0]))
		pmd.sequence = pmd.sequence[1:]
		if len(pmd.sequence) == 0 {
			log.Printf("empty value found for %s.", metName)
			toBeDeletedDataIndex = append(toBeDeletedDataIndex, i)
		}
	}
	hi.parsedMetricsData = deleteEntriesFromParsedMetricsData(toBeDeletedDataIndex, hi.parsedMetricsData)
}

var types map[string]metricsType

var counterExporters map[string]*counterExporter
var gaugeExporters map[string]*gaugeExporter
var histogramExporters map[string]*histogramExporter

var mu sync.Mutex

func init() {
	strToMetricsType = make(map[string]metricsType)
	strToMetricsType["counter"] = Counter
	strToMetricsType["gauge"] = Gauge
	strToMetricsType["histogram"] = Histogram

	types = make(map[string]metricsType)

	counterExporters = make(map[string]*counterExporter)
	gaugeExporters = make(map[string]*gaugeExporter)
	histogramExporters = make(map[string]*histogramExporter)
}

func parseSequence(sequence string) ([]float64, error) {
	result := make([]float64, 0)

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

			init, err := strconv.ParseFloat(initStr, 64)
			if err != nil {
				return nil, err
			}
			step, err := strconv.ParseFloat(stepStr, 64)
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
			val, err := strconv.ParseFloat(token, 64)
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
		if _, ok := histogramExporters[r.Spec.Name]; ok {
			return true, i
		}
	}
	return false, -1
}

func validBuckets(buckets []float64) bool {
	prev := -1.0
	for _, b := range buckets {
		if b <= 0 {
			return false
		}
		if b <= prev {
			return false
		}
		prev = b
	}
	return true
}

func validSpec(recipe []metricsRecipe) (bool, int) {
	for i, r := range recipe {
		if r.Spec.Name == "" {
			return false, i
		}
		if _, ok := strToMetricsType[r.Spec.Type]; !ok {
			return false, i
		}
		if len(r.Spec.Labels) == 0 {
			return false, i
		}
		if strToMetricsType[r.Spec.Type] == Histogram {
			if !validBuckets(r.Spec.Buckets) {
				return false, i
			}
		}
	}
	return true, -1
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

func ascending(sequence []float64) bool {
	prev := float64(0)
	for i, val := range sequence {
		if i != 0 && prev > val {
			return false
		}
		prev = val
	}
	return true
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

	if result, i := validSpec(recipe); !result {
		return fmt.Errorf("invalid metrics spec. name: %s, type: %s, labels: %v, buckets: %v",
			recipe[i].Spec.Name, recipe[i].Spec.Type, recipe[i].Spec.Labels, recipe[i].Spec.Buckets)
	}

	for _, r := range recipe {
		switch strToMetricsType[r.Spec.Type] {
		case Counter:
			exporter, err := newCounterExporter(&r)
			if err != nil {
				return err
			}
			counterExporters[r.Spec.Name] = exporter
		case Gauge:
			exporter, err := newGaugeExporter(&r)
			if err != nil {
				return err
			}
			gaugeExporters[r.Spec.Name] = exporter
		case Histogram:
			exporter, err := newHistogramExporter(&r)
			if err != nil {
				return err
			}
			histogramExporters[r.Spec.Name] = exporter
		default:
			panic(fmt.Sprintf("unknown type: %d", types[r.Spec.Name]))
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
		exporter.update(metName)
	}

	for metName, exporter := range gaugeExporters {
		exporter.update(metName)
	}

	for metName, exporter := range histogramExporters {
		exporter.update(metName)
	}
}

func Clear(force bool) {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("param: force=%v", force)

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

	for metName, exporter := range histogramExporters {
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
	case Histogram:
		if !prometheus.Unregister(histogramExporters[metricsName].histogramVec) {
			log.Printf("unregister failed. metricsName = %s", metricsName)
		}
		delete(histogramExporters, metricsName)
	default:
		panic(fmt.Sprintf("unknown type: %d", types[metricsName]))
	}
	delete(types, metricsName)

	log.Printf("metrics %v was removed", metricsName)
}
