import { createClient } from "@clickhouse/client-web";

// Configuration
const CHUNK_SIZE_HOURS = 6; // Process 6 hours at a time
const CHUNK_SIZE_MS = CHUNK_SIZE_HOURS * 60 * 60 * 1000;
const MIGRATION_START = new Date("2023-05-01T00:00:00Z");
const MIGRATION_END = new Date("2025-05-18T00:00:00Z");

async function main() {
  const ch = createClient({
    url: process.env.CLICKHOUSE_URL,

    clickhouse_settings: {
      output_format_json_quote_64bit_integers: 0,
      output_format_json_quote_64bit_floats: 0,
    },
  });

  if (!process.env.CLICKHOUSE_URL) {
    throw new Error("CLICKHOUSE_URL environment variable is required");
  }

  let end = MIGRATION_END.getTime();

  while (end > MIGRATION_START.getTime()) {
    const start = end - CHUNK_SIZE_MS;

    console.log(`⏳ Processing ${new Date(start).toLocaleString()} (${start})`);

    await Promise.all([
      ch.query({
        query: `
        INSERT INTO key_verifications_raw_v2
        SELECT
            request_id,
            time,
            workspace_id,
            key_space_id,
            identity_id,
            key_id,
            region,
            outcome,
            tags,
            0 as spent_credits,    -- v1 doesn't have this column, default to 0
            0.0 as latency         -- v1 doesn't have this column, default to 0.0
        FROM verifications.raw_key_verifications_v1
        WHERE time >= ${start}
          AND time < ${end};
        `,
      }),
      ch.query({
        query: `
        INSERT INTO ratelimits_raw_v2
        SELECT
          request_id,
          time,
          workspace_id,
          namespace_id,
          identifier,
          passed,
          0.0 as latency -- v1 doesn't have this column, default to 0.0
        FROM
          ratelimits.raw_ratelimits_v1
        WHERE time >= ${start}
          AND time < ${end};

        `,
      }),
      ch.query({
        query: `
        INSERT INTO api_requests_raw_v2
        SELECT
          request_id,
          time,
          workspace_id,
          host,
          method,
          path,
          request_headers,
          request_body,
          response_status,
          response_headers,
          response_body,
          error,
          service_latency,
          user_agent,
          ip_address,
          '' as region
        FROM
          metrics.raw_api_requests_v1
        WHERE time >= ${start}
          AND time < ${end};

        `,
      }),
    ]);

    end = start;
  }
}
main();
