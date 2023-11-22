type Memory = {
  // How many requests were made in the current window
  current: number;
  alarmScheduled?: number;
};

export class DurableObjectRatelimiter {
  private state: DurableObjectState;
  private memory: Memory;
  constructor(state: DurableObjectState) {
    this.state = state;
    this.state.blockConcurrencyWhile(async () => {
      const m = await this.state.storage.get<Memory>("rl");
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
    const req = (await request.json()) as { reset: number };

    this.memory.current += 1;

    if (!this.memory.alarmScheduled) {
      this.memory.alarmScheduled = req.reset;
      await this.state.storage.setAlarm(this.memory.alarmScheduled);
    }

    await this.state.storage.put("rl", this.memory);

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
