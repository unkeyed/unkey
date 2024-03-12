import { zValidator } from "@hono/zod-validator";
import { instrumentDO } from "@microlabs/otel-cf-workers";
import { Hono } from "hono";
import { z } from "zod";
import { traceConfig } from "../tracing/config";

type Memory = {
  current: number;
  alarmScheduled?: number;
};

class DO {
  private state: DurableObjectState;
  private memory: Memory;
  private readonly storageKey = "rl";
  private readonly hono = new Hono();
  constructor(state: DurableObjectState) {
    this.state = state;
    this.state.blockConcurrencyWhile(async () => {
      const m = await this.state.storage.get<Memory>(this.storageKey);
      if (m) {
        this.memory = m;
      }
    });
    this.memory ??= {
      current: 0,
    };

    this.hono.post(
      "/limit",
      zValidator("json", z.object({ reset: z.number().int() })),
      async (c) => {
        const { reset } = c.req.valid("json");
        this.memory.current += 1;

        if (!this.memory.alarmScheduled) {
          this.memory.alarmScheduled = reset;
          await this.state.storage.setAlarm(this.memory.alarmScheduled);
        }

        await this.state.storage.put(this.storageKey, this.memory);

        return c.json({
          current: this.memory.current,
        });
      },
    );
  }

  // Handle HTTP requests from clients.
  async fetch(request: Request) {
    return this.hono.fetch(request);
  }

  /**
   * alarm is called to clean up all state, which will remove the durable object from existence.
   */
  public async alarm(): Promise<void> {
    await this.state.storage.deleteAll();
  }
}

export const DurableObjectRatelimiter = instrumentDO(
  DO,
  traceConfig((env) => ({
    name: `api.${env.ENVIRONMENT}`,
    namespace: "DurableObjectRatelimiter",
    version: env.VERSION,
  })),
);
