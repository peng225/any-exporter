package exporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseValues(t *testing.T) {
	cases := []struct {
		desc         string
		values       string
		parsedValues []int
		isError      bool
	}{
		// Normal cases
		{
			desc:         "one element",
			values:       "1",
			parsedValues: []int{1},
			isError:      false,
		},
		{
			desc:         "three element",
			values:       "1 2 3",
			parsedValues: []int{1, 2, 3},
			isError:      false,
		},
		{
			desc:         "1+2x3 style",
			values:       "1+2x3",
			parsedValues: []int{1, 3, 5, 7},
			isError:      false,
		},
		{
			desc:         "1-2x3 style",
			values:       "1-2x3",
			parsedValues: []int{1, -1, -3, -5},
			isError:      false,
		},
		{
			desc:         "-1+2x3 style",
			values:       "-1+2x3",
			parsedValues: []int{-1, 1, 3, 5},
			isError:      false,
		},
		{
			desc:         "-1-2x3 style",
			values:       "-1-2x3",
			parsedValues: []int{-1, -3, -5, -7},
			isError:      false,
		},
		{
			desc:         "3x4 style",
			values:       "3x4",
			parsedValues: []int{3, 3, 3, 3, 3},
			isError:      false,
		},
		{
			desc:         "combination",
			values:       "1 2-3x4 1x2",
			parsedValues: []int{1, 2, -1, -4, -7, -10, 1, 1, 1},
			isError:      false,
		},
		// Error cases
		{
			desc:         "empty",
			values:       "",
			parsedValues: nil,
			isError:      true,
		},
		{
			desc:         "redundant +",
			values:       "+1+2x3",
			parsedValues: nil,
			isError:      true,
		},
		{
			desc:         "empty",
			values:       "",
			parsedValues: nil,
			isError:      true,
		},
		{
			desc:         "unnecessary space",
			values:       " 1+2x3",
			parsedValues: nil,
			isError:      true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.desc, func(t *testing.T) {
			parsedValues, err := parseValues(tt.values)
			if tt.isError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.parsedValues, parsedValues)
		})
	}
}
