package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	MAX_BUFFER_SIZE = 50000
	MAX_BATCH_SIZE  = 10000
	FLUSH_INTERVAL  = time.Second * 3
)

var (
	CLICKHOUSE_URL string
	BASIC_AUTH     string
	PORT           string
)

func init() {

	CLICKHOUSE_URL = os.Getenv("CLICKHOUSE_URL")
	if CLICKHOUSE_URL == "" {
		panic("CLICKHOUSE_URL must be defined")
	}
	BASIC_AUTH = os.Getenv("BASIC_AUTH")
	if BASIC_AUTH == "" {
		panic("BASIC_AUTH must be defined")
	}
	PORT = os.Getenv("PORT")
	if PORT == "" {
		PORT = "7123"
	}
}

type Batch struct {
	Rows   []string
	Params url.Values
}

func persist(batch *Batch) error {
	if len(batch.Rows) == 0 {
		return nil
	}

	u, err := url.Parse(CLICKHOUSE_URL)
	if err != nil {
		return err
	}

	u.RawQuery = batch.Params.Encode()

	req, err := http.NewRequest("POST", u.String(), strings.NewReader(strings.Join(batch.Rows, "\n")))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "text/plain")
	username := u.User.Username()
	password, ok := u.User.Password()
	if !ok {
		return fmt.Errorf("password not set")
	}
	req.SetBasicAuth(username, password)

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		log.Printf("GOLANG persisted %d rows for %s\n", len(batch.Rows), batch.Params.Get("query"))
	} else {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		fmt.Println("unable to persist", string(body))
	}
	return nil
}

func main() {
	requiredAuthorization := "Basic " + base64.StdEncoding.EncodeToString([]byte(BASIC_AUTH))

	buffer := make(chan *Batch, MAX_BUFFER_SIZE)
	// blocks until we've persisted everything and the process may stop
	done := make(chan bool)

	go func() {

		buffered := 0

		batchesByParams := make(map[string]*Batch)

		ticker := time.NewTicker(FLUSH_INTERVAL)

		flushAndReset := func() {
			for _, batch := range batchesByParams {
				err := persist(batch)
				if err != nil {
					log.Println("Error flushing:", err.Error())
				}
			}
			buffered = 0
			batchesByParams = make(map[string]*Batch)
			ticker.Reset(FLUSH_INTERVAL)
		}
		for {
			select {
			case b, ok := <-buffer:
				if !ok {
					// channel closed
					flushAndReset()
					done <- true
					return
				}

				params := b.Params.Encode()
				batch, ok := batchesByParams[params]
				if !ok {
					batchesByParams[params] = b
				} else {

					batch.Rows = append(batch.Rows, b.Rows...)
				}

				buffered += len(b.Rows)

				if buffered >= MAX_BATCH_SIZE {
					log.Println("Flushing due to max size")
					flushAndReset()
				}
			case <-ticker.C:
				log.Println("Flushing from ticker")

				flushAndReset()
			}
		}
	}()

	http.HandleFunc("/v1/liveness", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != requiredAuthorization {
			log.Println("invaldu authorization header, expected", requiredAuthorization, r.Header.Get("Authorization"))
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		query := r.URL.Query().Get("query")
		if query == "" || !strings.HasPrefix(strings.ToLower(query), "insert into") {
			http.Error(w, "wrong query", http.StatusBadRequest)
			return
		}

		params := r.URL.Query()
		params.Del("query_id")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read body", http.StatusInternalServerError)
		}
		rows := strings.Split(string(body), "\n")

		buffer <- &Batch{
			Params: params,
			Rows:   rows,
		}

		w.Write([]byte("ok"))
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fmt.Println("listening on", PORT)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", PORT), nil); err != nil {
		log.Fatalln("error starting server:", err)
	}

	<-ctx.Done()
	log.Println("shutting down")
	close(buffer)
	<-done

}
