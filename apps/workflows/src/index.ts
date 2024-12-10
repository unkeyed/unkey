import type { Env } from "./lib/env";

export { CountKeys } from "./workflows/count_keys_per_keyspace";
export { RefillRemaining } from "./workflows/refill_keys";

export default {
  async scheduled(event: ScheduledEvent, env: Env, _ctx: ExecutionContext) {
    console.info(event);
    switch (event.cron) {
      case "* * * * *": {
        const instance = await env.COUNT_KEYS.create();
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
