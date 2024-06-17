import type { Env } from "./env";
import type { Metrics } from "./metrics";

import { type Ratelimit, createRatelimitClient } from "@unkey/agent";

export function connectAgent(
  env: Required<Pick<Env, "AGENT_URL" | "AGENT_TOKEN">>,
  metrics?: Metrics,
): Ratelimit {
  const ratelimit = createRatelimitClient({
    baseUrl: env.AGENT_URL,
    token: env.AGENT_TOKEN,
  });
  if (!metrics) {
    return ratelimit;
  }

  return {
    liveness: async (...args: Parameters<Ratelimit["liveness"]>) => {
      const start = performance.now();
      const res = await ratelimit.liveness(...args);
      metrics.emit({
        metric: "metric.agent.latency",
        op: "liveness",
        latency: performance.now() - start,
      });
      return res;
    },
    ratelimit: async (...args: Parameters<Ratelimit["ratelimit"]>) => {
      const [req, opts] = args;
      const start = performance.now();
      const res = await ratelimit.ratelimit(req, {
        ...opts,
        headers: {
          ...opts?.headers,
          /*
           * Cloudflare's load balancer routes all requests with the same affinity id to the same endpoint
           */
          "Unkey-Session-Affinity-Id": req.identifier!,
        },
      });
      metrics.emit({
        metric: "metric.agent.latency",
        op: "ratelimit",
        latency: performance.now() - start,
      });
      return res;
    },
  };
}
