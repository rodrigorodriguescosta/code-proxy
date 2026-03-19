package account

import (
	"math"
	"time"
)

// CooldownForStatus returns the cooldown duration for an HTTP status code
func CooldownForStatus(httpStatus int, backoffLevel int) time.Duration {
	switch httpStatus {
	case 401:
		return 5 * time.Minute // Invalid token
	case 402:
		return 30 * time.Minute // Payment required
	case 403:
		return 30 * time.Minute // Forbidden
	case 429:
		return ExponentialBackoff(backoffLevel) // Rate limited
	case 500, 502:
		return 10 * time.Second // Server error
	case 503:
		return 30 * time.Second // Service unavailable
	case 504:
		return 10 * time.Second // Gateway timeout
	default:
		if httpStatus >= 400 {
			return 30 * time.Second
		}
		return 5 * time.Second
	}
}

// ExponentialBackoff calculates the wait time: 1s x 2^level, max 2min
func ExponentialBackoff(level int) time.Duration {
	if level < 0 {
		level = 0
	}
	d := time.Duration(math.Pow(2, float64(level))) * time.Second
	if d > 2*time.Minute {
		d = 2 * time.Minute
	}
	return d
}
