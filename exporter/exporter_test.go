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
		parsedSeq []int
		isError   bool
	}{
		// Normal cases
		{
			desc:      "one element",
			sequence:  "1",
			parsedSeq: []int{1},
			isError:   false,
		},
		{
			desc:      "three element",
			sequence:  "1 2 3",
			parsedSeq: []int{1, 2, 3},
			isError:   false,
		},
		{
			desc:      "1+2x3 style",
			sequence:  "1+2x3",
			parsedSeq: []int{1, 3, 5, 7},
			isError:   false,
		},
		{
			desc:      "1-2x3 style",
			sequence:  "1-2x3",
			parsedSeq: []int{1, -1, -3, -5},
			isError:   false,
		},
		{
			desc:      "-1+2x3 style",
			sequence:  "-1+2x3",
			parsedSeq: []int{-1, 1, 3, 5},
			isError:   false,
		},
		{
			desc:      "-1-2x3 style",
			sequence:  "-1-2x3",
			parsedSeq: []int{-1, -3, -5, -7},
			isError:   false,
		},
		{
			desc:      "3x4 style",
			sequence:  "3x4",
			parsedSeq: []int{3, 3, 3, 3, 3},
			isError:   false,
		},
		{
			desc:      "combination",
			sequence:  "1 2-3x4 1x2",
			parsedSeq: []int{1, 2, -1, -4, -7, -10, 1, 1, 1},
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
	}

	for _, tt := range cases {
		t.Run(tt.desc, func(t *testing.T) {
			parsedSeq, err := parseSequence(tt.sequence)
			if tt.isError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.parsedSeq, parsedSeq)
		})
	}
}

func TestDnvalidDataLabel(t *testing.T) {
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
