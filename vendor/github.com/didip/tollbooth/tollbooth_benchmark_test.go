package tollbooth

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/didip/tollbooth/config"
)

func BenchmarkLimitByKeys(b *testing.B) {
	limiter := config.NewLimiter(1, time.Second) // Only 1 request per second is allowed.

	for i := 0; i < b.N; i++ {
		LimitByKeys(limiter, []string{"127.0.0.1", "/"})
	}
}

func BenchmarkLimitByKeysWithExpiringBuckets(b *testing.B) {
	limiter := config.NewLimiterExpiringBuckets(1, time.Second, time.Minute, time.Minute) // Only 1 request per second is allowed.

	for i := 0; i < b.N; i++ {
		LimitByKeys(limiter, []string{"127.0.0.1", "/"})
	}
}

func BenchmarkBuildKeys(b *testing.B) {
	limiter := config.NewLimiter(1, time.Second)
	limiter.IPLookups = []string{"X-Real-IP", "RemoteAddr", "X-Forwarded-For"}
	limiter.Headers = make(map[string][]string)
	limiter.Headers["X-Real-IP"] = []string{"2601:7:1c82:4097:59a0:a80b:2841:b8c8"}

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		fmt.Printf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", limiter.Headers["X-Real-IP"][0])
	for i := 0; i < b.N; i++ {
		sliceKeys := BuildKeys(limiter, request)
		if len(sliceKeys) == 0 {
			fmt.Print("Length of sliceKeys should never be empty.")
		}
	}
}
