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
      const start = performance.now();

      const res = await fetch(`${opts.baseUrl}/ratelimit.v1.RatelimitService/Ratelimit`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${opts.token}`,
        },
        body: JSON.stringify(args[0]),
      });
      const body = await res.json<ReturnType<Ratelimit["ratelimit"]>>();

      // const [req, opts] = args;
      // const res = await ratelimit.ratelimit(req, {
      //   ...opts,
      //   headers: {
      //     ...opts?.headers,
      //     /*
      //      * Cloudflare's load balancer routes all requests with the same affinity id to the same endpoint
      //      */
      //     "Unkey-Session-Affinity-Id": req.identifier!,
      //   },
      // });
      metrics.emit({
        metric: "metric.agent.latency",
        op: "ratelimit",
        latency: performance.now() - start,
      });
      return body;
    },
    multiRatelimit: async (...args: Parameters<Ratelimit["multiRatelimit"]>) => {
      const start = performance.now();

      const res = await fetch(`${opts.baseUrl}/ratelimit.v1.RatelimitService/MultiRatelimit`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${opts.token}`,
        },
        body: JSON.stringify(args[0]),
      });
      const body = await res.json<ReturnType<Ratelimit["multiRatelimit"]>>();

      // const [req, opts] = args;
      // const res = await ratelimit.ratelimit(req, {
      //   ...opts,
      //   headers: {
      //     ...opts?.headers,
      //     /*
      //      * Cloudflare's load balancer routes all requests with the same affinity id to the same endpoint
      //      */
      //     "Unkey-Session-Affinity-Id": req.identifier!,
      //   },
      // });
      metrics.emit({
        metric: "metric.agent.latency",
        op: "multiRatelimit",
        latency: performance.now() - start,
      });
      return body;
    },
    pushPull: async (..._args: Parameters<Ratelimit["pushPull"]>) => {
      throw new Error("Not indented to be used");
    },
  };
}
