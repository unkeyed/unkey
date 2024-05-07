import { eq, schema } from "@unkey/db";
import { AesGCM } from "@unkey/encryption";
import type { Event } from "@unkey/events";
import { newId } from "@unkey/id";
import { createConnection } from "./pkg/db";
import { type Env, zEnv } from "./pkg/env";
import type { QueuePayload } from "./pkg/schema";
import { Analytics } from "./pkg/tinybird";

export default {
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

    const reporters = await db.query.verificationMonitors
      .findMany({
        where: (table, { lte }) => lte(table.nextExecution, event.scheduledTime),
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

      const eventId = newId("event");
      const now = new Date();

      const payload: Event = {
        type: "verifications.usage.record",
        timestamp: now.toISOString(),
        data: {
          eventId,
          interval: {
            start: query.start,
            end: query.end,
          },
          keySpaceId: r.keySpaceId,
          records: data.data,
        },
      };

      await env.WEBHOOKS_OUT.send({
        workspaceId: r.workspaceId,
        webhookId: r.webhookId,
        payload,
      });
      await db.insert(schema.events).values({
        id: eventId,
        workspaceId: r.workspaceId,
        webhookId: r.webhookId,
        time: now.getTime(),
        event: "verifications.usage.record",
        payload,
      });

      await db
        .update(schema.verificationMonitors)
        .set({
          highWaterMark: event.scheduledTime,
          nextExecution: event.scheduledTime + r.interval,
        })
        .where(eq(schema.verificationMonitors.id, r.id));
    }
  },

  async queue(batch: MessageBatch<QueuePayload>, rawEnv: Env): Promise<void> {
    const env = zEnv.parse(rawEnv);
    const db = createConnection(env);

    for (const message of batch.messages) {
      const deliveryId = newId("webhookDelivery");
      try {
        const webhook = await db.query.webhooks.findFirst({
          where: (table, { eq }) => eq(table.id, message.body.webhookId),
        });
        if (!webhook) {
          throw new Error(`webhook id not found: ${message.body.webhookId}`);
        }

        const encryptionKey = env.ENCRYPTION_KEYS.find(
          (k) => k.version === webhook.encryptionKeyVersion,
        );
        if (!encryptionKey) {
          throw new Error(`encrpytion key version missing: ${webhook.encryptionKeyVersion}`);
        }
        const aes = await AesGCM.withBase64Key(encryptionKey.key);

        const key = await aes.decrypt({
          iv: webhook.iv,
          ciphertext: webhook.ciphertext,
        });

        const time = Date.now();
        const res = await fetch(webhook.destination, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${key}`,
            "User-Agent": "unkey",
            "Webhook-Id": message.body.payload.data.eventId,
            "Webhook-Timestamp": Math.floor(
              new Date(message.body.payload.timestamp).getTime() / 1000,
            ).toString(),
          },
          body: JSON.stringify(message.body.payload),
          // signal: AbortSignal.timeout(5000) // timeout of 5s
        });

        const success = res.status >= 200 && res.status < 300;

        const delaySeconds = getRetryDelaySeconds(message.attempts);
        await db.insert(schema.deliveryAttempts).values({
          time,
          id: deliveryId,
          workspaceId: message.body.workspaceId,
          webhookId: message.body.webhookId,
          eventId: message.body.payload.data.eventId,
          attempt: message.attempts,
          nextAttemptAt: success ? null : Date.now() + delaySeconds * 1000,
          responseStatus: res.status,
          responseBody: (await res.text()).slice(0, 1000),
          success,
        });

        if (success) {
          message.ack();
          await db
            .update(schema.events)
            .set({
              state: "delivered",
            })
            .where(eq(schema.events.id, message.body.payload.data.eventId));
        } else if (message.attempts < 10) {
          await db
            .update(schema.events)
            .set({
              state: "retrying",
            })
            .where(eq(schema.events.id, message.body.payload.data.eventId));
          message.retry({
            delaySeconds: getRetryDelaySeconds(message.attempts),
          });
        } else {
          // fail it
          await db
            .update(schema.events)
            .set({
              state: "failed",
            })
            .where(eq(schema.events.id, message.body.payload.data.eventId));
          message.ack();
        }

        console.info(message);
      } catch (err) {
        console.error(err);
        const delaySeconds = getRetryDelaySeconds(message.attempts);
        await db.insert(schema.deliveryAttempts).values({
          id: deliveryId,
          time: Date.now(),
          workspaceId: message.body.workspaceId,
          webhookId: message.body.webhookId,
          eventId: message.body.payload.data.eventId,
          attempt: message.attempts,
          internalError: (err as Error).message,
          nextAttemptAt: Date.now() + delaySeconds * 1000,
          success: false,
        });
        await db
          .update(schema.events)
          .set({
            state: "retrying",
          })
          .where(eq(schema.events.id, message.body.payload.data.eventId));
        message.retry({ delaySeconds });
      }
    }
  },
};

/**
 * Return the delay in seconds until the next attempt
 *
 * Source: https://github.com/standard-webhooks/standard-webhooks/blob/main/spec/standard-webhooks.md#deliverability-and-reliability
 */
function getRetryDelaySeconds(currentAttempt: number): number {
  const delays = [
    0,
    5,
    5 * 60,
    30 * 60,
    2 * 60 * 60,
    5 * 60 * 60,
    10 * 60 * 60,
    14 * 60 * 60,
    20 * 60 * 60,
    24 * 60 * 60,
  ];

  return delays.at(currentAttempt) ?? 24 * 60 * 60;
}
