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

func TestBasicScenario(t *testing.T) {
	baseURL := "http://localhost:8080"
	f, err := os.Open("recipe.yaml")
	require.NoError(t, err)

	resp, err := http.Post(baseURL+"/recipe", "application/yaml", f)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	f.Close()

	// get metrics 1
	resp, err = http.Get(baseURL + "/metrics")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	metricsByte, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	metrics := string(metricsByte)
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val1"} 1`))
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val2"} 0`))
	assert.True(t, strings.Contains(metrics, `test2{aaa="aaa_val2",ccc="ccc_val1"} 0`))

	// get metrics 2
	resp, err = http.Get(baseURL + "/metrics")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	metricsByte, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	metrics = string(metricsByte)
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val1"} 3`))
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val2"} 1`))
	assert.True(t, strings.Contains(metrics, `test2{aaa="aaa_val2",ccc="ccc_val1"} 1`))

	// get metrics 3 (the value of test2 will be drained)
	resp, err = http.Get(baseURL + "/metrics")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	metricsByte, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	metrics = string(metricsByte)
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val1"} 6`))
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val2"} 2`))
	assert.True(t, strings.Contains(metrics, `test2{aaa="aaa_val2",ccc="ccc_val1"} 0`))

	// get metrics 4 (the value of test1{aaa="aaa_val1", bbb="bbb_val1"} will be drained)
	resp, err = http.Get(baseURL + "/metrics")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	metricsByte, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	metrics = string(metricsByte)
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val1"} 6`))
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val2"} 3`))
	assert.True(t, strings.Contains(metrics, `test2{aaa="aaa_val2",ccc="ccc_val1"} 0`))

	// delete recipe
	req, err := http.NewRequest(http.MethodDelete, baseURL+"/recipe", nil)
	require.NoError(t, err)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// get metrics 5 (test2 should already be deleted)
	resp, err = http.Get(baseURL + "/metrics")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	metricsByte, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	metrics = string(metricsByte)
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val1"} 6`))
	assert.True(t, strings.Contains(metrics, `test1{aaa="aaa_val1",bbb="bbb_val2"} 4`))
	assert.False(t, strings.Contains(metrics, "test2"))
}
