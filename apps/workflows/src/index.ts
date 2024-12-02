import type { Env } from "./lib/env";

export default {
  async scheduled(event: ScheduledEvent, env: Env, _ctx: ExecutionContext) {
    const instance = await env.REFILL_REMAINING.create();
    console.log(JSON.stringify({ event, instance }));
  },
};
