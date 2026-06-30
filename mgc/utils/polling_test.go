package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPollingInterval(t *testing.T) {
	def := 10 * time.Second

	testCases := []struct {
		name string
		env  string
		want time.Duration
	}{
		{name: "unset returns default", env: "", want: def},
		{name: "valid duration overrides default", env: "200ms", want: 200 * time.Millisecond},
		{name: "valid duration in seconds", env: "2s", want: 2 * time.Second},
		{name: "invalid duration returns default", env: "not-a-duration", want: def},
		{name: "negative duration returns default", env: "-5s", want: def},
		{name: "zero duration returns default", env: "0s", want: def},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv(PollingIntervalEnvVar, tc.env)
			}
			assert.Equal(t, tc.want, PollingInterval(def))
		})
	}
}
