package dev

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
)

func migrateClickhouse(ctx context.Context, dsn string) error {

	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return err
	}

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return err
	}

	defer conn.Close()

	err = conn.Ping(ctx)
	if err != nil {
		return err
	}

	return fs.WalkDir(schema.Migrations, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		f, err := schema.Migrations.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		content, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		queries := strings.Split(string(content), ";")

		log.Printf("Executing migration %s\n", path)
		for _, query := range queries {
			query = strings.TrimSpace(query)
			if query == "" {
				continue
			}

			err = conn.Exec(context.Background(), fmt.Sprintf("%s;", query))
			if err != nil {
				return err
			}
		}

		return nil
	})

}
