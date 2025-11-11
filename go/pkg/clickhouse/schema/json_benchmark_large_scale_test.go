//go:build goexperiment.jsonv2

package schema

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"fmt"
	"math/rand"
	"testing"

	"github.com/bytedance/sonic"
	goccyjson "github.com/goccy/go-json"
)

// Cached data
var cachedApiRequests = make(map[int][]ApiRequestV1)
var cachedApiRequestsJSON = make(map[int][]byte)

func getApiRequests(size int) []ApiRequestV1 {
	if cached, ok := cachedApiRequests[size]; ok {
		return cached
	}

	requests := make([]ApiRequestV1, size)
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	paths := []string{"/v2/keys/verifyKey", "/v2/keys/createKey", "/v2/ratelimit/limit"}

	for i := 0; i < size; i++ {
		requests[i] = ApiRequestV1{
			WorkspaceID:     fmt.Sprintf("ws_%d_%x", i, rand.Int63()),
			RequestID:       fmt.Sprintf("req_%d_%x", i, rand.Int63()),
			Time:            1699999999999 + int64(i*1000),
			Host:            "api.unkey.dev",
			Method:          methods[i%len(methods)],
			Path:            paths[i%len(paths)],
			RequestHeaders:  []string{fmt.Sprintf("Auth: Bearer sk_%d", i), "Content-Type: application/json"},
			RequestBody:     fmt.Sprintf(`{"key":"sk_%d","apiId":"api_%d"}`, i, i),
			ResponseStatus:  200 + (i % 5),
			ResponseHeaders: []string{"Content-Type: application/json"},
			ResponseBody:    fmt.Sprintf(`{"valid":true,"keyId":"key_%d"}`, i),
			ServiceLatency:  int64(10 + (i % 100)),
			UserAgent:       "unkey-go/1.0",
			IpAddress:       fmt.Sprintf("192.168.%d.%d", i/256, i%256),
			Country:         "US",
			City:            "San Francisco",
			Colo:            "SFO",
			Continent:       "NA",
		}
	}

	cachedApiRequests[size] = requests
	return requests
}

func getApiRequestsJSON(size int) []byte {
	if cached, ok := cachedApiRequestsJSON[size]; ok {
		return cached
	}

	data, _ := json.Marshal(getApiRequests(size))
	cachedApiRequestsJSON[size] = data
	return data
}

// Unmarshal benchmarks for all sizes
func BenchmarkUnmarshal_ApiRequest_1_StdlibV1(b *testing.B) {
	benchUnmarshal(b, 1, "stdlibv1")
}
func BenchmarkUnmarshal_ApiRequest_1_RealV2(b *testing.B) {
	benchUnmarshal(b, 1, "realv2")
}
func BenchmarkUnmarshal_ApiRequest_1_Sonic(b *testing.B) {
	benchUnmarshal(b, 1, "sonic")
}
func BenchmarkUnmarshal_ApiRequest_1_Goccy(b *testing.B) {
	benchUnmarshal(b, 1, "goccy")
}

func BenchmarkUnmarshal_ApiRequest_100_StdlibV1(b *testing.B) {
	benchUnmarshal(b, 100, "stdlibv1")
}
func BenchmarkUnmarshal_ApiRequest_100_RealV2(b *testing.B) {
	benchUnmarshal(b, 100, "realv2")
}

func BenchmarkUnmarshal_ApiRequest_100_Sonic(b *testing.B) {
	benchUnmarshal(b, 100, "sonic")
}
func BenchmarkUnmarshal_ApiRequest_100_Goccy(b *testing.B) {
	benchUnmarshal(b, 100, "goccy")
}

func BenchmarkUnmarshal_ApiRequest_10000_StdlibV1(b *testing.B) {
	benchUnmarshal(b, 10000, "stdlibv1")
}
func BenchmarkUnmarshal_ApiRequest_10000_RealV2(b *testing.B) {
	benchUnmarshal(b, 10000, "realv2")
}

func BenchmarkUnmarshal_ApiRequest_10000_Sonic(b *testing.B) {
	benchUnmarshal(b, 10000, "sonic")
}
func BenchmarkUnmarshal_ApiRequest_10000_Goccy(b *testing.B) {
	benchUnmarshal(b, 10000, "goccy")
}

func BenchmarkUnmarshal_ApiRequest_100000_StdlibV1(b *testing.B) {
	benchUnmarshal(b, 100000, "stdlibv1")
}
func BenchmarkUnmarshal_ApiRequest_100000_RealV2(b *testing.B) {
	benchUnmarshal(b, 100000, "realv2")
}

func BenchmarkUnmarshal_ApiRequest_100000_Sonic(b *testing.B) {
	benchUnmarshal(b, 100000, "sonic")
}
func BenchmarkUnmarshal_ApiRequest_100000_Goccy(b *testing.B) {
	benchUnmarshal(b, 100000, "goccy")
}

func BenchmarkUnmarshal_ApiRequest_1000000_StdlibV1(b *testing.B) {
	benchUnmarshal(b, 1000000, "stdlibv1")
}
func BenchmarkUnmarshal_ApiRequest_1000000_RealV2(b *testing.B) {
	benchUnmarshal(b, 1000000, "realv2")
}

func BenchmarkUnmarshal_ApiRequest_1000000_Sonic(b *testing.B) {
	benchUnmarshal(b, 1000000, "sonic")
}
func BenchmarkUnmarshal_ApiRequest_1000000_Goccy(b *testing.B) {
	benchUnmarshal(b, 1000000, "goccy")
}

// Marshal benchmarks
func BenchmarkMarshal_ApiRequest_1_StdlibV1(b *testing.B) {
	benchMarshal(b, 1, "stdlibv1")
}
func BenchmarkMarshal_ApiRequest_1_RealV2(b *testing.B) {
	benchMarshal(b, 1, "realv2")
}

func BenchmarkMarshal_ApiRequest_1_Sonic(b *testing.B) {
	benchMarshal(b, 1, "sonic")
}
func BenchmarkMarshal_ApiRequest_1_Goccy(b *testing.B) {
	benchMarshal(b, 1, "goccy")
}

func BenchmarkMarshal_ApiRequest_100_StdlibV1(b *testing.B) {
	benchMarshal(b, 100, "stdlibv1")
}
func BenchmarkMarshal_ApiRequest_100_RealV2(b *testing.B) {
	benchMarshal(b, 100, "realv2")
}

func BenchmarkMarshal_ApiRequest_100_Sonic(b *testing.B) {
	benchMarshal(b, 100, "sonic")
}
func BenchmarkMarshal_ApiRequest_100_Goccy(b *testing.B) {
	benchMarshal(b, 100, "goccy")
}

func BenchmarkMarshal_ApiRequest_10000_StdlibV1(b *testing.B) {
	benchMarshal(b, 10000, "stdlibv1")
}
func BenchmarkMarshal_ApiRequest_10000_RealV2(b *testing.B) {
	benchMarshal(b, 10000, "realv2")
}

func BenchmarkMarshal_ApiRequest_10000_Sonic(b *testing.B) {
	benchMarshal(b, 10000, "sonic")
}
func BenchmarkMarshal_ApiRequest_10000_Goccy(b *testing.B) {
	benchMarshal(b, 10000, "goccy")
}

func BenchmarkMarshal_ApiRequest_100000_StdlibV1(b *testing.B) {
	benchMarshal(b, 100000, "stdlibv1")
}
func BenchmarkMarshal_ApiRequest_100000_RealV2(b *testing.B) {
	benchMarshal(b, 100000, "realv2")
}

func BenchmarkMarshal_ApiRequest_100000_Sonic(b *testing.B) {
	benchMarshal(b, 100000, "sonic")
}
func BenchmarkMarshal_ApiRequest_100000_Goccy(b *testing.B) {
	benchMarshal(b, 100000, "goccy")
}

func BenchmarkMarshal_ApiRequest_1000000_StdlibV1(b *testing.B) {
	benchMarshal(b, 1000000, "stdlibv1")
}
func BenchmarkMarshal_ApiRequest_1000000_RealV2(b *testing.B) {
	benchMarshal(b, 1000000, "realv2")
}

func BenchmarkMarshal_ApiRequest_1000000_Sonic(b *testing.B) {
	benchMarshal(b, 1000000, "sonic")
}
func BenchmarkMarshal_ApiRequest_1000000_Goccy(b *testing.B) {
	benchMarshal(b, 1000000, "goccy")
}

// Helper functions
func benchUnmarshal(b *testing.B, size int, library string) {
	data := getApiRequestsJSON(size)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	switch library {
	case "stdlibv1":
		for i := 0; i < b.N; i++ {
			var reqs []ApiRequestV1
			if err := json.Unmarshal(data, &reqs); err != nil {
				b.Fatal(err)
			}
		}
	case "realv2":
		for i := 0; i < b.N; i++ {
			var reqs []ApiRequestV1
			if err := jsonv2.Unmarshal(data, &reqs); err != nil {
				b.Fatal(err)
			}
		}
	case "sonic":
		for i := 0; i < b.N; i++ {
			var reqs []ApiRequestV1
			if err := sonic.Unmarshal(data, &reqs); err != nil {
				b.Fatal(err)
			}
		}
	case "goccy":
		for i := 0; i < b.N; i++ {
			var reqs []ApiRequestV1
			if err := goccyjson.Unmarshal(data, &reqs); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func benchMarshal(b *testing.B, size int, library string) {
	reqs := getApiRequests(size)
	b.ReportAllocs()

	switch library {
	case "stdlibv1":
		for i := 0; i < b.N; i++ {
			data, err := json.Marshal(reqs)
			if err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(data)))
		}
	case "realv2":
		for i := 0; i < b.N; i++ {
			data, err := jsonv2.Marshal(reqs)
			if err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(data)))
		}

	case "sonic":
		for i := 0; i < b.N; i++ {
			data, err := sonic.Marshal(reqs)
			if err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(data)))
		}
	case "goccy":
		for i := 0; i < b.N; i++ {
			data, err := goccyjson.Marshal(reqs)
			if err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(data)))
		}
	}
}
