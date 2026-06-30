package utils

import (
	"os"
	"time"
)

// PollingIntervalEnvVar overrides the interval between status checks when
// waiting for long-running operations. It accepts Go duration strings
// (e.g. "200ms", "5s") and exists mainly so acceptance tests against a
// fake API can run fast.
const PollingIntervalEnvVar = "MGC_POLLING_INTERVAL"

// PollingInterval returns the polling interval configured via
// MGC_POLLING_INTERVAL, falling back to def when the variable is unset,
// empty, invalid or non-positive.
func PollingInterval(def time.Duration) time.Duration {
	v := os.Getenv(PollingIntervalEnvVar)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return def
	}
	return d
}
