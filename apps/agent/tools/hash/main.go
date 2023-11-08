package main

import (
	"encoding/json"
	"fmt"

	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
)

var keys = []string{
	"3ZiQHfeoXjmfE8yS9tHEJuD5",
	"3ZSBKYgk6KzQ45RxUsGy6r9e",
	"3ZNr9iFJhD57ataQqsDNDWC3",
	"3ZY3Mc1PQsqnFVUaXrJSa8Bw",
	"3Zf6Q3NH2jobAqL8j8K7k4rL",
	"3Zkwyg7Pgbi4UG2AYyhzJbvQ",
	"3Zjw5PTJDhaDDmB6mqKBPeLS",
	"3ZidmMM7JzaYKZiG6WM2FUMc",
	"3ZS8Dw92w8RLKn98p2DPerEH",
	"3ZWdo1EqYkgUJhjw9DVnt2Ux",
	"3ZjKpTvqbdfkn5gEbR5xNQ4V",
	"3ZNTiKjoe5uJfetcUVsSy9iv",
	"3ZNS9EE8CoQwJSa2ADZprVAn",
	"3Zc9YsWmjoNucft5djDEUgi7",
	"3Zc7X3PAMUVsYevmHsMoVCQo",
	"3ZdYWR6P83ScEHRr8H7zNL7F",
	"3ZGztph421F62oySLZVVhkHZ",
	"3ZKfh45FBBEvnMBKRpaiNAJk",
	"3ZeZMhDdezrqQj4qJ8ymR6PC",
	"3ZSi6CkUC4QNqqWpzdyDnQpW",
	"3ZP6eYqV9sTgffTCyA6ocrtJ",
	"3ZNGtqbZV1kbts5RZskCZXZW",
	"3ZJ2mYd8QeaxtrvRzV9R9ip5",
	"3ZTixMPMKrCzAXLxEYYTdy8G",
	"3ZfAvhmc23rcE3HZjEZDFrLh",
	"3ZkSDFm3RL9YB29aespSJUXT",
	"3ZUyGsaZQg5x6qd22ADHR8jM",
	"3Zjcq6uRvzfqFxJwV4KCGNHG",
	"3Zm4bZt84WbaAK7M8xwoUL4W",
	"3ZbGmmSZmAVm9hGzkPXEFEms",
	"3ZGrs4nA2Qcat7ECQaFvc7nx",
	"3ZgRgpqZBH4k2ovVoN2hevxn",
	"3ZNnK92Ce1Lsazdu8oxPdXQb",
	"3ZMq8gqEp2fopgqSYBNk1e3J",
	"3ZmXehZeiHqy3p7dY6mw18Dj",
	"3ZdEKYyKwaYsjiLATyo4a3dS",
	"3ZfeWuEiGgEqn4PykPFj6VLg",
	"3ZixamEfv22xU6tQkR8BvPjd",
	"3ZZkwfxa2QcGxnAgEjdhvbG5",
	"3ZTBxqD3xPPg4cSHje8XHawh",
	"3ZivV6rEZ9WeKkHj4PWCwmjh",
	"3Za1xerUXy95rTMc687i27F8",
	"3Zm9xJrtvv1SbLxSVv9JC3Yh",
	"3ZWZ9sAiK3WhxXefxk83raYv",
	"3Zj78ZXozBEqNiuvoCUYujVd",
	"3ZjJ1tHrWNcQWJtBJLRPN7m1",
	"3ZGgtK9yv8ek3BxfzzbwR34M",
	"3ZabUV64TGJnEQphTQwie8qq",
	"3ZihERXAbDBbhcYao5Av2FuT",
	"3ZK7oHJUG1Y1PdPuDcRHoFM4",
	"3ZbKbeM29ZFdGm9ByvdKnZdP",
	"3ZSra7KqtwrdgmJ5hxXmPLQj",
	"3ZVvrgJUMg41sP5AxKnMZ9eP",
	"3ZcFC7zNWQYaLRN4W8KMkmU6",
	"3ZjSC8wGXuuohFFdbiaJRfN1",
	"3ZNTqBBvGgSfUan4gnjfKQ7z",
	"3Zk9jeZLLCbnjg18bQ6yRNfL",
	"3ZkWJ2b9n813EEMfsU9AVZQS",
	"3ZhT2dfhxQhbqLbdGrPE24q2",
	"3ZWzMtc2onktoGTmGGiKaseM",
	"3ZQgCSPGEJEuGL27jEbWhh68",
	"3ZYBwwFDvyNcETEibaSRUHJ7",
	"3ZepohbaFi9uNBSWRRaRdF9A",
	"3ZYKpPWPD9JpX4kjsyubW8v4",
	"3ZJZ1YrcjXHBMuBJdS2nBAxq",
	"3ZHxhRWe6gTMjcdtc7pAFhxH",
	"3ZXzXGe36q16WaudkbwR3fXg",
	"3ZkwAQosEzE7pj9G42ddFhoj",
	"3ZdPFPKVsmnqNn1Kup4Xx1Md",
	"3ZFeSzZi4bAHLaADiPyJy7BN",
	"3ZUqVSVmQLwaJ2CuU1dJDazQ",
	"3ZnJpPhMA2vD1u8HcfGYMV1i",
	"3ZUb5Gqdxx6R1eRcJhAt5cpW",
	"3Zgwzd17XhEaVBUm1GtYx2dS",
	"3ZXk7N4MqzScryq3HUorGVoT",
	"3ZNsjFHHgp3ic1WLC1vuHvQe",
	"3ZRje4duYPhbtmL6pXmW3nvi",
	"3ZPStw9cqGcZMEFdoNa7ByHj",
	"3ZhG4Ly6BGGvdswxVia84Dzh",
	"3ZHN1DGSNye9anQNUdMqppJe",
	"3ZG2fq6wALyJywNrL8Ao21FV",
	"3ZV7d2rPcL5bAwrN6Ge4M8wa",
	"3ZJqdBEVAdKw6r79Rw8RaXUN",
	"3ZUxddz9c7JUL2KN1fTFQZxK",
	"3ZNHvujebbtoxtE79qkykqvA",
	"3ZSuJVGzVNDGzWaWYbaCNgaj",
	"3ZZXNksZ4RXnwUAmAn41bZ2z",
	"3ZZYJ6wk7pDbV9tC89PMmuRb",
	"3ZKVoGXFtDLh1ok8zB2VjkmY",
	"3ZXruFjgmgKqHRWV2drSrQEP",
	"3Zcjv9de13Vhjz3suRdi4vP8",
	"3ZJvdobi31Uz8sfQVFdhwhtV",
	"3ZZBnzgMBhBMpNKkWJEhrRtq",
	"3ZSZKE8z43BPUesHULeyawwq",
	"3ZaLMgGSJ9zzwnKnxHhmz4QV",
	"3ZTGzRYdtc6WVV5BCsPon8ao",
	"3ZQWWTR7sE2k6sBRt4W78PpD",
	"3ZaKQi84aYnYyZGqtgsb1Ezp",
	"3Zc4GbbdTGCWFLDPsPA4NZ5b",
	"3ZfF1GqgWK1LKEqQnQ49rvzF",
}

// Return the hash of the key used for authentication
func main() {
	m := make(map[string]string)
	for _, k := range keys {
		m[k] = hash.Sha256(k)
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))

}
