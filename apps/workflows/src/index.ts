import type { Env } from "./lib/env";

export default {
  async scheduled(event: ScheduledEvent, env: Env, _ctx: ExecutionContext) {
    console.info(event);
    switch (event.cron) {
      case "*/5 * * * *": {
        const instance = await env.REFILL_REMAINING.create();
        console.info(JSON.stringify({ event, instance }));

        break;
      }
      case "0 0 * * *": {
        const instance = await env.REFILL_REMAINING.create();
        console.info(JSON.stringify({ event, instance }));
        break;
      }
    }
  },
};
