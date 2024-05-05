import { eq, schema } from "@unkey/db";
import { createConnection } from "./pkg/db";
import { type Env, zEnv } from "./pkg/env";
import type { QueuePayload } from "./pkg/schema";
import { Analytics } from "./pkg/tinybird";
export default {
  // Our fetch handler is invoked on a HTTP request: we can send a message to a queue
  // during (or after) a request.
  // https://developers.cloudflare.com/queues/platform/javascript-apis/#producer
  async scheduled(event: ScheduledEvent, rawEnv: Env, _ctx: ExecutionContext) {
    console.info("running", JSON.stringify(event));
    const envCheck = zEnv.safeParse(rawEnv);
    if (!envCheck.success) {
      console.error("ENV ERROR", envCheck.error.message);
      throw new Error(envCheck.error.message);
    }

    const env = envCheck.data;
    const db = createConnection(env);
    const tb = new Analytics({ tinybirdToken: env.TINYBIRD_TOKEN });

    const reporters = await db.query.usageReporters
      .findMany({
        where: (table, { lte, and, isNull }) =>
          and(isNull(table.deletedAt), lte(table.nextExecution, event.scheduledTime)),
        with: {
          webhook: true,
        },
      })
      .catch((err) => {
        console.error(err);
        throw err;
      });

    console.info("running", reporters.length, "reporters");

    for (const r of reporters) {
      console.info(r);
      const query = {
        workspaceId: r.workspaceId,
        keySpaceId: r.keySpaceId,
        start: r.highWaterMark ?? 0,
        end: event.scheduledTime,
      };
      console.info(query);
      const data = await tb.getVerificationsByOwnerId(query);

      console.info("tinybird data", JSON.stringify(data.data, null, 2));

      await env.WEBHOOKS_OUT.send({
        payload: {
          type: "unkey.usage.record",
          timestamp: new Date().toISOString(),
          data: {
            interval: {
              start: query.start,
              end: query.end,
            },
            keySpaceId: r.keySpaceId,
            records: data.data,
          },
        },
      });
      await db
        .update(schema.usageReporters)
        .set({
          highWaterMark: event.scheduledTime,
          nextExecution: event.scheduledTime + r.interval,
        })
        .where(eq(schema.usageReporters.id, r.id));
    }
  },

  async queue(batch: MessageBatch<QueuePayload>, rawEnv: Env): Promise<void> {
    const _env = zEnv.parse(rawEnv);

    for (const message of batch.messages) {
      console.info(message);
    }
  },
};
