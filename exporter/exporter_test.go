package exporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSequence(t *testing.T) {
	cases := []struct {
		desc      string
		sequence  string
		parsedSeq []float64
		isError   bool
	}{
		// Normal cases
		{
			desc:      "one element",
			sequence:  "1",
			parsedSeq: []float64{1},
			isError:   false,
		},
		{
			desc:      "three element",
			sequence:  "1 2 3",
			parsedSeq: []float64{1, 2, 3},
			isError:   false,
		},
		{
			desc:      "1+2x3 style",
			sequence:  "1+2x3",
			parsedSeq: []float64{1, 3, 5, 7},
			isError:   false,
		},
		{
			desc:      "1-2x3 style",
			sequence:  "1-2x3",
			parsedSeq: []float64{1, -1, -3, -5},
			isError:   false,
		},
		{
			desc:      "-1+2x3 style",
			sequence:  "-1+2x3",
			parsedSeq: []float64{-1, 1, 3, 5},
			isError:   false,
		},
		{
			desc:      "-1-2x3 style",
			sequence:  "-1-2x3",
			parsedSeq: []float64{-1, -3, -5, -7},
			isError:   false,
		},
		{
			desc:      "3x4 style",
			sequence:  "3x4",
			parsedSeq: []float64{3, 3, 3, 3, 3},
			isError:   false,
		},
		{
			desc:      "combination",
			sequence:  "1 2-3x4 1x2",
			parsedSeq: []float64{1, 2, -1, -4, -7, -10, 1, 1, 1},
			isError:   false,
		},
		{
			desc:      "float combination",
			sequence:  "1.2 3.4-5.6x3 1.1x2",
			parsedSeq: []float64{1.2, 3.4, -2.2, -7.8, -13.4, 1.1, 1.1, 1.1},
			isError:   false,
		},
		// Error cases
		{
			desc:      "empty",
			sequence:  "",
			parsedSeq: nil,
			isError:   true,
		},
		{
			desc:      "redundant +",
			sequence:  "+1+2x3",
			parsedSeq: nil,
			isError:   true,
		},
		{
			desc:      "empty",
			sequence:  "",
			parsedSeq: nil,
			isError:   true,
		},
		{
			desc:      "unnecessary space",
			sequence:  " 1+2x3",
			parsedSeq: nil,
			isError:   true,
		},
		{
			desc:      "float times",
			sequence:  "1+2x3.4",
			parsedSeq: nil,
			isError:   true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.desc, func(t *testing.T) {
			parsedSeq, err := parseSequence(tt.sequence)
			if tt.isError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.InDeltaSlice(t, tt.parsedSeq, parsedSeq, 0.001)
		})
	}
}

func TestInvalidDataLabel(t *testing.T) {
	specLabel := []string{"aaa", "bbb"}

	cases := []struct {
		desc           string
		dataLabel      map[string]string
		expectedResult bool
	}{
		{
			desc: "success",
			dataLabel: map[string]string{
				"aaa": "foo",
				"bbb": "var",
			},
			expectedResult: false,
		},
		{
			desc: "lack of label",
			dataLabel: map[string]string{
				"aaa": "foo",
			},
			expectedResult: true,
		},
		{
			desc: "extra label",
			dataLabel: map[string]string{
				"aaa": "foo",
				"bbb": "var",
				"ccc": "baz",
			},
			expectedResult: true,
		},
		{
			desc: "wrong label",
			dataLabel: map[string]string{
				"aaa":  "foo",
				"bbbb": "var",
			},
			expectedResult: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.desc, func(t *testing.T) {
			result := invalidDataLabel(specLabel, tt.dataLabel)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestValidBuckets(t *testing.T) {
	cases := []struct {
		desc           string
		buckets        []float64
		expectedResult bool
	}{
		{
			desc:           "strictly increasing from a positive number",
			buckets:        []float64{1, 2.5, 5, 7.5},
			expectedResult: true,
		},
		{
			desc:           "single value",
			buckets:        []float64{0.5},
			expectedResult: true,
		},
		{
			desc:           "descending order",
			buckets:        []float64{2, 3, 5, 4, 7},
			expectedResult: false,
		},
		{
			desc:           "starts with zero",
			buckets:        []float64{0, 1, 2},
			expectedResult: false,
		},
		{
			desc:           "starts with a negative value",
			buckets:        []float64{-1, 1, 2},
			expectedResult: false,
		},
		{
			desc:           "same value",
			buckets:        []float64{1, 2, 2, 3},
			expectedResult: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.desc, func(t *testing.T) {
			result := validBuckets(tt.buckets)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
