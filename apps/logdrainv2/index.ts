import { Axiom } from "@axiomhq/js";
import { decompressSync, strFromU8 } from "fflate";
import { z } from "zod";

import { serve } from "@hono/node-server";
import { Hono } from "hono";

const axiom = new Axiom({
  token: process.env.AXIOM_TOKEN!,
  orgId: "unkey-hsbi",
});

const logsSchema = z.array(
  z
    .object({
      TimestampMs: z.number(),
      Level: z.string(),
      Message: z.array(
        z.string().transform((s) => {
          try {
            return JSON.parse(s);
          } catch (err) {
            console.error((err as Error).message, s);
            return {};
          }
        }),
      ),
    })
    .passthrough(),
);

const fetchSchema = z.object({
  Event: z
    .object({
      RayID: z.string(),
      Request: z.object({
        Method: z.string(),
        URL: z.string(),
      }),
      Response: z.object({
        Status: z.number(),
      }),
    })
    .passthrough(),
  EventTimestampMs: z.number(),
  EventType: z.literal("fetch"),
  Exceptions: z.array(z.object({}).passthrough()),
  Logs: logsSchema,
  Outcome: z.string(),
  ScriptName: z.string(),
  ScriptTags: z.array(z.object({}).passthrough()),
});

const alarmSchema = z.object({
  Event: z.object({
    ScheduledTimeMs: z.number(),
  }),
  EventTimestampMs: z.number(),
  EventType: z.literal("alarm"),
  Exceptions: z.array(z.object({}).passthrough()),
  Logs: logsSchema,
  Outcome: z.string(),
  ScriptName: z.string(),
  ScriptTags: z.array(z.object({}).passthrough()),
});

const eventSchema = z.discriminatedUnion("EventType", [fetchSchema, alarmSchema]);

setInterval(() => {
  console.log("I'm still alive");
}, 5000);

const app = new Hono({});
app.all("*", async (c) => {
  console.log("incoming request", c.req.url);
  try {
    const b = await c.req.blob();

    const buf = await b.arrayBuffer();

    const dec = decompressSync(new Uint8Array(buf));
    const str = strFromU8(dec);
    const rawLines = str.split("\n").filter((l) => l.trim().length > 0);

    const lines = rawLines
      .map((l) => {
        try {
          return eventSchema.parse(JSON.parse(l));
        } catch (err) {
          console.error((err as Error).message, l);
          return null;
        }
      })
      .filter((l) => l !== null) as Array<z.infer<typeof eventSchema>>;

    console.log("received", lines.length, "lines");

    const now = Date.now();
    axiom.ingest(
      "logdrain-lag",
      lines.map((l) => ({
        eventTime: l.EventTimestampMs,
        logdrainTime: now,
        latency: now - l.EventTimestampMs,
      })),
    );
    axiom.ingest(
      "playing-with-logdrains",
      lines.map((l) => ({
        ...l,
      })),
    );
    for (const line of lines) {
      for (const log of line.Logs) {
        for (const message of log.Message) {
          axiom.ingest("logdrain-logs", {
            rayId: "RayID" in line.Event ? line.Event.RayID : null,
            time: log.TimestampMs,
            level: log.Level,
            message,
          });
        }
      }
    }

    axiom.ingest("logdrain", {
      level: "info",
      message: `ingested ${lines.length} events`,
      events: lines.length,
    });
    await axiom.flush();
    return c.json({ url: c.req.url });
  } catch (e) {
    const err = e as Error;
    console.error(err.message);

    axiom.ingest("logdrain", {
      level: "error",
      message: err.message,
    });
    await axiom.flush();
    return new Response(err.message, { status: 500 });
  }
});
const port = process.env.PORT ?? 8000;

const srv = serve({
  fetch: app.fetch,
  port: Number(port),
});

srv.on("listening", () => {
  console.log("listening", port);
});

srv.on("close", () => {
  console.log("closing");
});
