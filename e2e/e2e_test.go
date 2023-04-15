package e2e

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL = "http://localhost:8080"
)

func cleanUp(t *testing.T) {
	t.Helper()

	// force delete recipe
	req, err := http.NewRequest(http.MethodDelete, baseURL+"/recipe", nil)
	require.NoError(t, err)
	q := req.URL.Query()
	q.Add("force", "true")
	req.URL.RawQuery = q.Encode()
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func getMetrics(t *testing.T) string {
	t.Helper()

	resp, err := http.Get(baseURL + "/metrics")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	metricsByte, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	return string(metricsByte)
}

func postMetrics(t *testing.T, recipeFileName string, expectedStatus int) {
	t.Helper()

	f, err := os.Open(recipeFileName)
	require.NoError(t, err)
	defer f.Close()

	resp, err := http.Post(baseURL+"/recipe", "application/yaml", f)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode)
}

func TestCounterAndGauge(t *testing.T) {
	// post metrics recipe
	postMetrics(t, "counter-and-gauge.yaml", http.StatusOK)

	// conflict recipe post
	postMetrics(t, "counter-and-gauge.yaml", http.StatusConflict)

	// get metrics 1
	metrics := getMetrics(t)
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val1"} 1`))
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val2"} 0`))
	assert.True(t, strings.Contains(metrics, `test2{aaa="aaa_val2",ccc="ccc_val1"} 0`))

	// get metrics 2
	metrics = getMetrics(t)
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val1"} 2`))
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val2"} 1`))
	assert.True(t, strings.Contains(metrics, `test2{aaa="aaa_val2",ccc="ccc_val1"} 1`))

	// get metrics 3 (the value of test2 will be drained)
	metrics = getMetrics(t)
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val1"} 3`))
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val2"} 1`))
	assert.True(t, strings.Contains(metrics, `test2{aaa="aaa_val2",ccc="ccc_val1"} 0`))

	// get metrics 4 (the value of test1{aaa="aaa_val1", bbb="bbb_val1"} will be drained)
	metrics = getMetrics(t)
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val1"} 3`))
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val2"} 1`))
	assert.True(t, strings.Contains(metrics, `test2{aaa="aaa_val2",ccc="ccc_val1"} 0`))

	// delete recipe
	req, err := http.NewRequest(http.MethodDelete, baseURL+"/recipe", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// get metrics 5 (test2 should already be deleted)
	metrics = getMetrics(t)
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val1"} 3`))
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val2"} 1`))
	assert.False(t, strings.Contains(metrics, "test2"))

	// force delete recipe
	req, err = http.NewRequest(http.MethodDelete, baseURL+"/recipe", nil)
	require.NoError(t, err)
	q := req.URL.Query()
	q.Add("force", "true")
	req.URL.RawQuery = q.Encode()
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// post again
	postMetrics(t, "counter-and-gauge.yaml", http.StatusOK)

	cleanUp(t)
}

func TestHistogram(t *testing.T) {
	// post metrics recipe
	postMetrics(t, "histogram.yaml", http.StatusOK)

	// conflict recipe post
	postMetrics(t, "histogram.yaml", http.StatusConflict)

	// get metrics 1
	metrics := getMetrics(t)
	assert.True(t, strings.Contains(metrics, `test3_bucket{ccc="ccc_val1",ddd="ddd_val1",le="0.5"} 0`), metrics)
	assert.True(t, strings.Contains(metrics, `test3_bucket{ccc="ccc_val1",ddd="ddd_val1",le="1"} 1`), metrics)
	assert.True(t, strings.Contains(metrics, `test3_bucket{ccc="ccc_val2",ddd="ddd_val2",le="0.5"} 1`), metrics)

	// get metrics 2
	metrics = getMetrics(t)
	assert.True(t, strings.Contains(metrics, `test3_bucket{ccc="ccc_val1",ddd="ddd_val1",le="1"} 1`), metrics)
	assert.True(t, strings.Contains(metrics, `test3_bucket{ccc="ccc_val1",ddd="ddd_val1",le="2"} 2`), metrics)
	assert.True(t, strings.Contains(metrics, `test3_bucket{ccc="ccc_val2",ddd="ddd_val2",le="2"} 1`), metrics)
	assert.True(t, strings.Contains(metrics, `test3_bucket{ccc="ccc_val2",ddd="ddd_val2",le="4"} 2`), metrics)

	// get metrics 3
	metrics = getMetrics(t)
	assert.True(t, strings.Contains(metrics, `test3_bucket{ccc="ccc_val1",ddd="ddd_val1",le="32"} 2`), metrics)
	assert.True(t, strings.Contains(metrics, `test3_bucket{ccc="ccc_val1",ddd="ddd_val1",le="+Inf"} 3`), metrics)
	assert.True(t, strings.Contains(metrics, `test3_bucket{ccc="ccc_val2",ddd="ddd_val2",le="4"} 2`), metrics)
	assert.True(t, strings.Contains(metrics, `test3_bucket{ccc="ccc_val2",ddd="ddd_val2",le="8"} 3`), metrics)

	cleanUp(t)
}
