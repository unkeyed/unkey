import { Err, Ok, type Result } from "@unkey/error";
import type { Logger } from "@unkey/worker-logging";
import { cloudflareRatelimiter } from "../env";
import type { Context } from "../hono/app";
import type { Metrics } from "../metrics";
import { retry } from "../util/retry";
import { Agent } from "./agent";
import {
  type RateLimiter,
  RatelimitError,
  type RatelimitRequest,
  type RatelimitResponse,
} from "./interface";
export class AgentRatelimiter implements RateLimiter {
  private readonly logger: Logger;
  private readonly metrics: Metrics;
  private readonly cache: Map<string, { reset: number; current: number }>;
  private readonly agent: Agent;
  constructor(opts: {
    agent: { url: string; token: string };
    logger: Logger;
    metrics: Metrics;
    cache: Map<string, { reset: number; current: number }>;
  }) {
    this.logger = opts.logger;
    this.metrics = opts.metrics;
    this.cache = opts.cache;
    this.agent = new Agent(opts.agent.url, opts.agent.token, this.metrics, this.logger);
  }

  private getId(req: RatelimitRequest): string {
    const now = Date.now();
    const window = Math.floor(now / req.interval);

    return [req.identifier, window, req.shard].join("::");
  }

  private setCacheMax(id: string, current: number, reset: number) {
    const maxEntries = 10_000;
    this.metrics.emit({
      metric: "metric.cache.size",
      name: "ratelimitcache",
      tier: "memory",
      size: this.cache.size,
    });
    if (this.cache.size > maxEntries) {
      const now = Date.now();
      for (const [k, v] of this.cache) {
        if (this.cache.size <= maxEntries) {
          break;
        }
        if (v.reset < now) {
          this.cache.delete(k);
        }
      }
    }
    const cached = this.cache.get(id) ?? { reset: 0, current: 0 };
    if (current > cached.current) {
      this.cache.set(id, { reset, current });
      return current;
    }
  }

  public async limit(
    c: Context,
    req: RatelimitRequest,
  ): Promise<Result<RatelimitResponse, RatelimitError>> {
    const start = performance.now();
    try {
      if (req.async) {
        // Construct a binding key that could match a configured ratelimiter
        const lookup = `RL_${req.limit}_${Math.round(req.interval / 1000)}s` as keyof typeof c.env;
        const binding = c.env[lookup];

        if (binding) {
          const res = await cloudflareRatelimiter.parse(binding).limit({ key: req.identifier });

          this.metrics.emit({
            metric: "metric.ratelimit",
            workspaceId: req.workspaceId,
            namespaceId: req.namespaceId,
            latency: performance.now() - start,
            identifier: req.identifier,
            mode: "async",
            error: false,
            success: res.success,
            source: "cloudflare",
          });
          return Ok({
            passed: res.success,
            reset: -1,
            current: -1,
            remaining: -1,
            triggered: res.success ? null : req.name,
          });
        }
      }
    } catch (err) {
      this.logger.error("cfrl failed, falling back to agent", {
        error: (err as Error).message,
      });
    }
    const res = await this._limit(c, req);
    this.metrics.emit({
      metric: "metric.ratelimit",
      workspaceId: req.workspaceId,
      namespaceId: req.namespaceId,
      latency: performance.now() - start,
      identifier: req.identifier,
      mode: req.async ? "async" : "sync",
      error: !!res.err,
      success: res?.val?.passed,
      source: "agent",
    });
    return res;
  }

  /**
   * Do not use
   */
  public async multiLimit(
    c: Context,
    req: Array<RatelimitRequest>,
  ): Promise<Result<RatelimitResponse, RatelimitError>> {
    const res = await Promise.all(req.map((r) => this.limit(c, r)));
    for (const r of res) {
      if (r.err) {
        return r;
      }
      if (!r.val.passed) {
        return r;
      }
    }
    if (res.length > 0) {
      // biome-ignore lint/style/noNonNullAssertion: Safe to leave
      return Ok(res[0].val!);
    }

    return Ok({
      current: -1,
      passed: true,
      reset: -1,
      remaining: -1,
      triggered: null,
    });
  }

  private async _limit(
    c: Context,
    req: RatelimitRequest,
  ): Promise<Result<RatelimitResponse, RatelimitError>> {
    const window = Math.floor(Date.now() / req.interval);
    const reset = (window + 1) * req.interval;
    const cost = req.cost ?? 1;
    const id = this.getId(req);

    /**
     * Catching identifiers that exceeded the limit already
     *
     * This might not happen too often, but in extreme cases the cache should hit and we can skip
     * the request to the durable object entirely, which speeds everything up and is cheaper for us
     */
    const cached = this.cache.get(id) ?? { current: 0, reset: 0 };
    if (cached.current >= req.limit) {
      return Ok({
        passed: false,
        current: cached.current,
        reset,
        remaining: 0,
        triggered: req.name,
      });
    }

    const p = retry(3, async () =>
      this.callAgent(c, {
        requestId: c.get("requestId"),
        identifier: req.identifier,
        cost,
        duration: req.interval,
        limit: req.limit,
        name: req.name,
      }).catch((err) => {
        this.logger.error("error calling agent", {
          error: err.message,
          json: JSON.stringify(err),
        });
        throw err;
      }),
    );

    // A rollout of the sync rate limiting
    // Isolates younger than 60s must not sync. It would cause a stampede of requests as the cache is entirely empty
    const isolateCreatedAt = c.get("isolateCreatedAt");
    const isOlderThan60s = isolateCreatedAt ? Date.now() - isolateCreatedAt > 60_000 : false;
    const shouldSyncOnNoData = isOlderThan60s && Math.random() < c.env.SYNC_RATELIMIT_ON_NO_DATA;
    const cacheHit = this.cache.has(id);
    const sync = !req.async || (!cacheHit && shouldSyncOnNoData);
    if (sync) {
      const res = await p;
      if (res.val) {
        this.setCacheMax(id, res.val.current, res.val.reset);
      }
      return res;
    }

    c.executionCtx.waitUntil(
      p.then(async (res) => {
        if (res.err) {
          this.logger.error(res.err.message);
          return;
        }
        this.setCacheMax(id, res.val.current, res.val.reset);

        this.metrics.emit({
          workspaceId: req.workspaceId,
          metric: "metric.ratelimit.accuracy",
          identifier: req.identifier,
          namespaceId: req.namespaceId,
          responded: cached.current + cost <= req.limit,
          correct: res.val.current + cost <= req.limit,
        });
        await this.metrics.flush();
      }),
    );
    if (cached.current + cost > req.limit) {
      return Ok({
        current: cached.current,
        passed: false,
        reset,
        remaining: req.limit - cached.current,
        triggered: req.name,
      });
    }
    cached.current += cost;
    this.setCacheMax(id, cached.current, reset);

    return Ok({
      passed: true,
      current: cached.current,
      reset,
      remaining: req.limit - cached.current,
      triggered: null,
    });
  }

  private async callAgent(
    c: Context,
    req: {
      requestId: string;
      identifier: string;
      duration: number;
      cost: number;
      limit: number;
      name: string;
    },
  ): Promise<Result<RatelimitResponse, RatelimitError>> {
    try {
      let res: Awaited<ReturnType<Agent["ratelimit"]>> | undefined = undefined;
      let err: Error | undefined = undefined;
      const rlRequest = {
        identifier: req.identifier,
        limit: req.limit,
        duration: req.duration,
        cost: req.cost,
        name: req.name,
      };
      for (let i = 0; i <= 3; i++) {
        try {
          res = await this.agent.ratelimit(c, rlRequest);

          break;
        } catch (e) {
          this.logger.warn("calling the agent for ratelimiting failed, retrying ...", {
            identifier: req.identifier,
            error: (e as Error).message,
            attempt: i + 1,
          });
          err = e as Error;
        }
      }
      if (!res) {
        this.logger.error("calling the agent for ratelimiting failed", {
          identifier: req.identifier,
          error: err?.message,
        });
        return Err(new RatelimitError({ message: err?.message ?? "ratelimit failed" }));
      }

      return Ok({
        current: Number(res.limit - res.remaining),
        reset: Number(res.reset),
        passed: res.success,
        remaining: Number(res.remaining),
        triggered: res.success ? null : req.name,
      });
    } catch (e) {
      const err = e as Error;
      this.logger.error("ratelimit failed", {
        identifier: req.identifier,
        error: err.message,
        stack: err.stack,
        cause: err.cause,
      });
      return Err(new RatelimitError({ message: err.message }));
    }
  }
}
