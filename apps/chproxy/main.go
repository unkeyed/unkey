package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
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
	logger         *slog.Logger
)

func setupLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false,
	})

	logger = slog.New(handler)

	slog.SetDefault(logger)

	logger.Info("chproxy starting",
		slog.Int("max_buffer_size", MAX_BUFFER_SIZE),
		slog.Int("max_batch_size", MAX_BATCH_SIZE),
		slog.String("flush_interval", FLUSH_INTERVAL.String()))
}

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
		logger.Info("rows persisted",
			slog.Int("count", len(batch.Rows)),
			slog.String("query", batch.Params.Get("query")))
	} else {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		logger.Error("unable to persist rows",
			slog.String("response", string(body)),
			slog.Int("status_code", res.StatusCode),
			slog.String("query", batch.Params.Get("query")))
	}
	return nil
}

func main() {
	setupLogger()

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
					logger.Error("error flushing batch",
						slog.String("error", err.Error()),
						slog.String("query", batch.Params.Get("query")))
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
					logger.Info("flushing due to max size")
					flushAndReset()
				}
			case <-ticker.C:
				logger.Info("flushing from ticker")

				flushAndReset()
			}
		}
	}()

	http.HandleFunc("/v1/liveness", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != requiredAuthorization {
			logger.Warn("invalid authorization header",
				slog.String("expected", requiredAuthorization),
				slog.String("authorization", r.Header.Get("Authorization")))
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
			logger.Error("failed to read request body", slog.String("error", err.Error()))
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

	srv := &http.Server{Addr: fmt.Sprintf(":%s", PORT)}
	go func() {
		logger.Info("listening", slog.String("port", PORT))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("failed to start server", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	close(buffer)
	<-done

	logger.Info("shutdown complete")
}
