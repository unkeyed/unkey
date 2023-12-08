import { z } from "zod";

type Memory = {
  current: number;
  alarmScheduled?: number;
};

export class DurableObjectRatelimiter {
  private state: DurableObjectState;
  private memory: Memory;
  private readonly storageKey = "rl";
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
  }

  // Handle HTTP requests from clients.
  async fetch(request: Request) {
    const req = z
      .object({
        reset: z.number().int(),
      })
      .safeParse(await request.json());
    if (!req.success) {
      console.error("invalid DO req", req.error.message);
      return Response.json({
        current: 0,
      });
    }

    this.memory.current += 1;

    if (!this.memory.alarmScheduled) {
      this.memory.alarmScheduled = req.data.reset;
      await this.state.storage.setAlarm(this.memory.alarmScheduled);
    }

    await this.state.storage.put(this.storageKey, this.memory);

    return Response.json({
      current: this.memory.current,
    });
  }

  /**
   * alarm is called to clean up all state, which will remove the durable object from existence.
   */
  public async alarm(): Promise<void> {
    await this.state.storage.deleteAll();
  }
}
