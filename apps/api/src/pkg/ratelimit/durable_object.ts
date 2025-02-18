import { zValidator } from "@hono/zod-validator";
import { Hono } from "hono";
import { z } from "zod";

type Memory = {
  current: number;
  alarmScheduled?: number;
};

export class DurableObjectRatelimiter {
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
      zValidator(
        "json",
        z.object({
          reset: z.number().int(),
          cost: z.number().int().default(1),
          limit: z.number().int(),
        }),
      ),
      async (c) => {
        const { reset, cost, limit } = c.req.valid("json");
        if (!this.memory.alarmScheduled) {
          this.memory.alarmScheduled = reset;
          await this.state.storage.setAlarm(this.memory.alarmScheduled);
        }
        if (this.memory.current + cost > limit) {
          return c.json({
            success: false,
            current: this.memory.current,
          });
        }
        this.memory.current += cost;

        await this.state.storage.put(this.storageKey, this.memory);

        return c.json({
          success: true,
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
