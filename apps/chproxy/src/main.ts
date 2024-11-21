import { z } from "zod";

const MAX_BATCH_SIZE = 10000;
const MAX_BATCH_TIME = 3000; // milliseconds

type Batch = {
  createdAt: number;
  rows: string[];
  params: URLSearchParams;
};

const env = z
  .object({
    // url including basic auth
    // ie: https://user:password@xxx.clickhouse.cloud
    CLICKHOUSE_URL: z.string(),
    PORT: z
      .string()
      .optional()
      .default("7123")
      .transform((s) => Number.parseInt(s)),
    // user:pass
    // We receive a base64 of this in the authorization header ie: `Basic ZGVmYXVsdDo=`
    BASIC_AUTH: z.string(),
  })
  .parse(process.env);

const requiredAuthorization = `Basic ${btoa(env.BASIC_AUTH)}`;

const buffer = new Map<string, Batch>();

async function flush(force?: boolean): Promise<void> {
  const now = Date.now();

  for (const [key, batch] of buffer.entries()) {
    if (force || now >= batch.createdAt + MAX_BATCH_TIME) {
      await persist(key);
    }
  }
}

// persist inserts the data into clickhouse and removes it from the buffer
async function persist(key: string): Promise<void> {
  const batch = buffer.get(key);
  if (!batch) {
    return;
  }

  const url = new URL(env.CLICKHOUSE_URL);
  batch.params.forEach((v, k) => {
    url.searchParams.set(k, v);
  });

  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "text/plain",
      Authorization: `Basic ${btoa([url.username, url.password].join(":"))}`,
    },
    body: batch.rows.join("\n"),
  });
  if (res.ok) {
    buffer.delete(key);
    console.info(`persisted ${batch.rows.length} rows`);
  } else {
    console.error("unable to persist", await res.text(), JSON.stringify(batch));
  }
}

setInterval(flush, 1000);

const server = Bun.serve({
  port: env.PORT,
  fetch: async (req: Request): Promise<Response> => {
    const url = new URL(req.url);

    if (url.pathname === "/v1/liveness") {
      return new Response("I'm alive");
    }

    if (req.headers.get("Authorization") !== requiredAuthorization) {
      return new Response("unauthorized", { status: 401 });
    }

    const query = url.searchParams.get("query");
    if (!query || !query.startsWith("INSERT INTO")) {
      return new Response("wrong query", { status: 400 });
    }

    const params = url.searchParams;
    params.delete("query_id");
    params.sort();

    const key = params.toString();

    const rows = (await req.text()).split("\n");

    const existing = buffer.get(key);
    if (existing) {
      const size = existing.rows.push(...rows);
      if (size >= MAX_BATCH_SIZE) {
        await persist(key);
      }
    } else {
      buffer.set(key, {
        createdAt: Date.now(),
        params,
        rows,
      });
    }

    return new Response("ok");
  },
  error: (err) => {
    console.error(err);
    return new Response("internal server error", { status: 500 });
  },
});

console.info("listening on", server.hostname, server.port);
process.on("SIGTERM", async (s) => {
  console.warn("Received signal", s);

  server.stop();
  await flush(true);
  process.exit(0);
});
