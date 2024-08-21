package identities

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1RatelimitRatelimit "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_ratelimit_ratelimit"

	"github.com/unkeyed/unkey/apps/agent/pkg/uid"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

func TestRatelimitsAccuracy(t *testing.T) {

	request := v1RatelimitRatelimit.V1RatelimitRatelimitRequest{}
	request.Body.Identifier = uid.New("test")
	request.Body.Limit = 100
	request.Body.Duration = time.Minute.Milliseconds()

	b, err := json.Marshal(request.Body)
	require.NoError(t, err)

	rate := vegeta.Rate{Freq: 100, Per: time.Second}
	testDuration := 5 * time.Minute

	targets := []vegeta.Target{{
		Method: "POST",
		URL:    fmt.Sprintf("%s/ratelimit.v1.RatelimitService/Ratelimit", "http://localhost:8080"),
		Header: http.Header{
			"Authorization": []string{"Bearer agent-auth-secret"},
			"Content-Type":  []string{"application/json"},
		},
		Body: b,
	}}

	attacker := vegeta.NewAttacker()

	passed := 0
	rejected := 0
	errors := 0

	for res := range attacker.Attack(vegeta.NewStaticTargeter(targets...), rate, testDuration, "XX") {
		t.Log(string(res.Body))
		body := v1RatelimitRatelimit.V1RatelimitRatelimitResponse{}
		err := json.Unmarshal(res.Body, &body.Body)
		if err != nil {
			errors++
			continue
		}
		if body.Body.Success {
			passed++
		} else {
			rejected++
		}
	}
	t.Logf("passed: %d, rejected: %d, total: %d", passed, rejected, passed+rejected)

	require.LessOrEqual(t, errors, 5)
	require.LessOrEqual(t, passed, 1500)

}
