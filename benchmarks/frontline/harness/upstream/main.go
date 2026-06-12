// Command upstream is the benchmark upstream: a minimal HTTP server standing
// in for a deployment instance. It mirrors the upstream used in the original
// frontline comparison runs: a 38-byte JSON body at "/" and a 16 KiB body at
// "/16k", draining request bodies so proxied uploads never stall.
//
// It runs as its own OS process so the proxy under test never shares a
// runtime, scheduler, or GC with it.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	port := flag.Int("port", 38080, "port to listen on")
	flag.Parse()

	small := []byte(`{"message":"hello from the bench"}` + "\n")    // 35 bytes + padding below
	small = append(small, []byte("ok\n")...)                        // 38 bytes total
	big := bytes.Repeat([]byte("x"), 16<<10)                        // 16 KiB

	mux := http.NewServeMux()
	mux.HandleFunc("/16k", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(big)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(small)
	})

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	log.Printf("upstream listening on %s", addr)
	//nolint:gosec // benchmark tool, no timeouts wanted
	log.Fatal(http.ListenAndServe(addr, mux))
}
