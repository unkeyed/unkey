import { createClient } from "@clickhouse/client-web";

const now = Date.now() + 6 * 60 * 60 * 1000;

const tables = [
  {
    name: "default.key_verifications_per_minute",
    dt: 60 * 60 * 1000,
    start: now - 32 * 24 * 60 * 60 * 1000,
    end: now,
  },
  {
    name: "default.key_verifications_per_hour",
    dt: 60 * 60 * 1000,
    start: now - 40 * 24 * 60 * 60 * 1000,
    end: now,
  },
  {
    name: "default.key_verifications_per_day",
    dt: 23 * 60 * 60 * 1000,
    start: now - 100 * 24 * 60 * 60 * 1000,
    end: now,
  },
  {
    name: "default.key_verifications_per_month",
    dt: 12 * 24 * 60 * 60 * 1000,
    start: now - 4 * 356 * 24 * 60 * 60 * 1000,
    end: now,
  },
];

if (!process.env.CLICKHOUSE_URL) {
  throw new Error("CLICKHOUSE_URL is not set");
}

const rawCH = createClient({
  url: process.env.CLICKHOUSE_URL,

  clickhouse_settings: {
    output_format_json_quote_64bit_integers: 0,
    output_format_json_quote_64bit_floats: 0,
    http_send_timeout: 60000,
  },
});

const semaphore = new Map<string, Promise<void>>();
let concurrency = 1;

for (const { name, dt, start, end } of tables) {
  const v2 = `${name}_v2`;
  const v3 = `${name}_v3`;

  console.info("start", start, "end", end);

  for (let t = start; t < end; t += dt) {
    console.log(
      `${name}: ${new Date(t).toLocaleString("de")} - ${new Date(
        t + dt
      ).toLocaleString("de")}`
    );

    const res = await rawCH.query({
      query: `
        SELECT DISTINCT key_id
        FROM ${v2}
        WHERE time >= fromUnixTimestamp64Milli(${t})
        AND time < fromUnixTimestamp64Milli(${t + dt})
        AND not startsWith(key_id, 'test_')
      `,
    });
    const json = (await res.json()) as {
      data: Array<{ key_id: string }>;
    };
    console.log(json);

    const keyIds = json.data.map(({ key_id }) => key_id);
    for (let i = 0; i < keyIds.length; i++) {
      const keyId = keyIds[i];
      const semKey = `${name}-${t}-${keyId}`;

      while (semaphore.size >= concurrency) {
        await Promise.race(semaphore.values());
      }

      console.log(
        semKey,
        `${i}/${keyIds.length} - ${semaphore.size} / ${Math.floor(concurrency)}`
      );

      semaphore.set(
        semKey,
        rawCH
          .query({
            query: `
            INSERT INTO ${v3}
            SELECT
              time,
              workspace_id,
              key_space_id,
              identity_id,
              external_id,
              key_id,
              outcome,
              tags,
              count,
              spent_credits,
              latency_avg,
              latency_p75,
              latency_p99
            FROM ${v2}
            WHERE time >= fromUnixTimestamp64Milli(${t})
            AND time < fromUnixTimestamp64Milli(${t + dt})
            AND key_id = '${keyId}'
          `,
          })
          .then(() => {
            concurrency = Math.min(100, concurrency + 1 / concurrency);
          })
          .catch(async (err) => {
            console.error(err.message);
            concurrency = Math.max(1, concurrency / 2);
            await new Promise((resolve) => setTimeout(resolve, 10000));
          })
          .finally(() => {
            semaphore.delete(semKey);
          })
      );
    }
  }
}

await Promise.all(semaphore.values());
