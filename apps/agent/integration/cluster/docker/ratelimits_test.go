package identities

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/gen/openapi"

	"github.com/unkeyed/unkey/apps/agent/pkg/uid"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

func TestRatelimitsAccuracy(t *testing.T) {

	baseURL := os.Getenv("AGENT_BASE_URL")
	require.NotEmpty(t, baseURL, "AGENT_BASE_URL must be set")

	testCases := []struct {
		limit        int64
		duration     time.Duration
		rate         vegeta.Rate
		testDuration time.Duration
	}{

		{
			limit:        5000,
			duration:     2 * time.Minute,
			rate:         vegeta.Rate{Freq: 1000, Per: 10 * time.Second},
			testDuration: 5 * time.Minute,
		},
		{
			limit:        10,
			duration:     1 * time.Minute,
			rate:         vegeta.Rate{Freq: 100, Per: time.Second},
			testDuration: 3 * time.Minute,
		},
		{
			limit:        500,
			duration:     1 * time.Minute,
			rate:         vegeta.Rate{Freq: 10, Per: time.Second},
			testDuration: 5 * time.Minute,
		},
		{
			limit:        100,
			duration:     10 * time.Second,
			rate:         vegeta.Rate{Freq: 50, Per: time.Second},
			testDuration: 1 * time.Minute,
		},
		{
			limit:        500,
			duration:     10 * time.Minute,
			rate:         vegeta.Rate{Freq: 1, Per: time.Second},
			testDuration: 10 * time.Minute,
		},
		{
			limit:        1,
			duration:     1 * time.Minute,
			rate:         vegeta.Rate{Freq: 1, Per: time.Second},
			testDuration: 1 * time.Minute,
		},
		{
			limit:        1000,
			duration:     10 * time.Minute,
			rate:         vegeta.Rate{Freq: 5, Per: time.Second},
			testDuration: 10 * time.Minute,
		},
		{
			limit:        50,
			duration:     1 * time.Minute,
			rate:         vegeta.Rate{Freq: 200, Per: time.Second},
			testDuration: 2 * time.Minute,
		},
		{
			limit:        5000,
			duration:     10 * time.Minute,
			rate:         vegeta.Rate{Freq: 100, Per: time.Second},
			testDuration: 10 * time.Minute,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("[%d/%s] attacked with %s over %s", tc.limit, tc.duration, tc.rate.String(), tc.testDuration), func(t *testing.T) {
			t.Parallel()
			request := openapi.V1RatelimitRatelimitRequestBody{
				Identifier: uid.New("test"),
				Limit:      tc.limit,
				Duration:   tc.duration.Milliseconds(),
			}

			b, err := json.Marshal(request)
			require.NoError(t, err)

			target := vegeta.Target{
				Method: "POST",
				URL:    fmt.Sprintf("%s/ratelimit.v1.RatelimitService/Ratelimit", baseURL),
				Header: http.Header{
					"Authorization": []string{"Bearer agent-auth-secret"},
					"Content-Type":  []string{"application/json"},
				},
				Body: b,
			}

			attacker := vegeta.NewAttacker()

			total := int64(0)
			passed := int64(0)
			rejected := int64(0)
			errors := int64(0)

			for res := range attacker.Attack(vegeta.NewStaticTargeter(target), tc.rate, tc.testDuration, "v1.ratelimit.Ratelimit") {
				total++
				body := openapi.V1RatelimitRatelimitResponseBody{}
				err := json.Unmarshal(res.Body, &body)
				if err != nil {
					errors++
					continue
				}

				if body.Success {
					passed++
				} else {
					rejected++
				}
				// t.Logf("res: %+v", body.Body)
			}
			t.Logf("passed: %d, rejected: %d, total: %d", passed, rejected, passed+rejected)
			requestsSent := int64(tc.rate.Freq) * int64(tc.testDuration/tc.rate.Per)
			require.Equal(t, total, requestsSent)

			fullWindows := int64(math.Floor(float64(tc.testDuration.Milliseconds()) / float64(tc.duration.Milliseconds())))
			upperLimit := int64(float64((fullWindows+1)*tc.limit) * 2)
			lowerLimit := int64(float64(fullWindows*tc.limit) * 0.8)
			if requestsSent < lowerLimit {
				lowerLimit = requestsSent
			}

			require.LessOrEqual(t, errors, int64(5))
			require.GreaterOrEqual(t, passed, lowerLimit)
			require.LessOrEqual(t, passed, upperLimit)
		})
	}

}
