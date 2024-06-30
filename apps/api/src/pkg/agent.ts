import type { Metrics } from "./metrics";

import { type Ratelimit, createRatelimitClient } from "@unkey/agent";
export type { RatelimitResponse, Ratelimit } from "@unkey/agent";

export { protoInt64 } from "@unkey/agent";

export function connectAgent(
  opts: { baseUrl: string; token: string },
  metrics?: Metrics,
): Ratelimit {
  const ratelimit = createRatelimitClient(opts);
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
    multiRatelimit: async (...args: Parameters<Ratelimit["multiRatelimit"]>) => {
      const [req] = args;
      const start = performance.now();
      const res = await ratelimit.multiRatelimit(req);
      metrics.emit({
        metric: "metric.agent.latency",
        op: "multiRatelimit",
        latency: performance.now() - start,
      });
      return res;
    },
    pushPull: async (..._args: Parameters<Ratelimit["pushPull"]>) => {
      throw new Error("Not indented to be used");
    },
  };
}
