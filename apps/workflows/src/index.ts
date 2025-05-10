import type { Env } from "./lib/env";

export { Invoicing } from "./workflows/invoicing";

export default {
  async scheduled(event: ScheduledEvent, env: Env, _ctx: ExecutionContext) {
    console.info(event);
    switch (event.cron) {
      case "0 13 1 * *": {
        const instance = await env.INVOICING.create();
        console.info(JSON.stringify({ event, instance }));
        break;
      }
    }
  },
};
