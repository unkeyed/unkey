import { Err, Ok, type Result, SchemaError } from "@unkey/error";
import type { Logger } from "@unkey/worker-logging";
import { z } from "zod";
import type { Metrics } from "../metrics";

import type { Context } from "../hono/app";
import { Agent } from "./agent";
import {
  type RateLimiter,
  RatelimitError,
  type RatelimitRequest,
  type RatelimitResponse,
} from "./interface";

export class DurableRateLimiter implements RateLimiter {
  private readonly namespace: DurableObjectNamespace;
  private readonly domain: string;
  private readonly logger: Logger;
  private readonly metrics: Metrics;
  private readonly cache: Map<string, number>;
  private readonly agent?: Agent;
  constructor(opts: {
    namespace: DurableObjectNamespace;
    agent?: { url: string; token: string };
    domain?: string;
    logger: Logger;
    metrics: Metrics;
    cache: Map<string, number>;
  }) {
    this.namespace = opts.namespace;
    this.domain = opts.domain ?? "unkey.dev";
    this.logger = opts.logger;
    this.metrics = opts.metrics;
    this.cache = opts.cache;
    if (opts.agent) {
      this.agent = new Agent(opts.agent.url, opts.agent.token, this.metrics);
    }
  }

  private getId(req: RatelimitRequest): string {
    const now = Date.now();
    const window = Math.floor(now / req.interval);

    return [req.identifier, window, req.shard].join("::");
  }

  private setCacheMax(id: string, i: number): number {
    const current = this.cache.get(id) ?? 0;
    if (i > current) {
      this.cache.set(id, i);
      return i;
    }
    return current;
  }

  public async limit(
    c: Context,
    req: RatelimitRequest,
  ): Promise<Result<RatelimitResponse, RatelimitError>> {
    const start = performance.now();

    const res = await this._limit(c, req);
    this.metrics.emit({
      metric: "metric.ratelimit",
      workspaceId: req.workspaceId,
      namespaceId: req.namespaceId,
      latency: performance.now() - start,
      identifier: req.identifier,
      mode: req.async ? "async" : "sync",
      error: !!res.err,
      success: res?.val?.pass,
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
      if (!r.val.pass) {
        return r;
      }
    }
    if (res.length > 0) {
      return Ok(res[0].val!);
    }

    return Ok({
      current: -1,
      pass: true,
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
     * the request to the durable object entirely, which speeds everything up and is cheper for us
     */
    let current = this.cache.get(id) ?? 0;
    if (current >= req.limit) {
      return Ok({
        pass: false,
        current,
        reset,
        remaining: 0,
        triggered: req.name,
      });
    }

    const p = this.agent
      ? (async () => {
          const a = await this.callAgent(c, {
            requestId: c.get("requestId"),
            identifier: req.identifier,
            cost,
            duration: req.interval,
            limit: req.limit,
            name: req.name,
          });
          if (a.err) {
            this.logger.error("error calling agent", {
              error: a.err.message,
              json: JSON.stringify(a.err),
            });
            return this.callDurableObject({
              requestId: c.get("requestId"),
              identifier: req.identifier,
              objectName: id,
              window,
              reset,
              cost,
              limit: req.limit,
              name: req.name,
            });
          }
          return a;
        })()
      : this.callDurableObject({
          requestId: c.get("requestId"),
          identifier: req.identifier,
          objectName: id,
          window,
          reset,
          cost,
          limit: req.limit,
          name: req.name,
        });

    if (!req.async) {
      const res = await p;
      if (res.val) {
        this.setCacheMax(id, res.val.current);
      }
      return res;
    }

    c.executionCtx.waitUntil(
      p.then(async (res) => {
        if (res.err) {
          console.error(res.err.message);
          return;
        }
        this.setCacheMax(id, res.val.current);

        this.metrics.emit({
          workspaceId: req.workspaceId,
          metric: "metric.ratelimit.accuracy",
          identifier: req.identifier,
          namespaceId: req.namespaceId,
          responded: current + cost <= req.limit,
          correct: res.val.current + cost <= req.limit,
        });
        await this.metrics.flush();
      }),
    );
    if (current + cost > req.limit) {
      return Ok({
        current,
        pass: false,
        reset,
        remaining: req.limit - current,
        triggered: req.name,
      });
    }
    current += cost;
    this.cache.set(id, current);

    return Ok({
      pass: true,
      current,
      reset,
      remaining: req.limit - current,
      triggered: null,
    });
  }

  private getStub(name: string): DurableObjectStub {
    return this.namespace.get(this.namespace.idFromName(name));
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
          res = await this.agent!.ratelimit(c, rlRequest);

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
        pass: res.success,
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

  private async callDurableObject(req: {
    requestId: string;
    identifier: string;
    objectName: string;
    window: number;
    reset: number;
    cost: number;
    limit: number;
    name: string;
  }): Promise<Result<RatelimitResponse, RatelimitError>> {
    try {
      const url = `https://${this.domain}/limit`;

      // try twice
      let res: Response | undefined = undefined;
      let err: Error | undefined = undefined;
      for (let i = 0; i <= 3; i++) {
        try {
          res = await this.getStub(req.objectName).fetch(url, {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              "Unkey-Request-Id": req.requestId,
            },
            body: JSON.stringify({
              reset: req.reset,
              cost: req.cost,
              limit: req.limit,
            }),
          });
          break;
        } catch (e) {
          this.logger.warn("calling the ratelimit DO failed, retrying ...", {
            identifier: req.identifier,
            error: (e as Error).message,
            attempt: i + 1,
          });
          err = e as Error;
        }
      }
      if (!res) {
        this.logger.error("calling the ratelimit DO failed", {
          identifier: req.identifier,
          error: err?.message,
        });
        return Err(new RatelimitError({ message: err?.message ?? "ratelimit failed" }));
      }

      const json = await res.json();
      const parsed = z.object({ current: z.number(), success: z.boolean() }).safeParse(json);
      if (!parsed.success) {
        return Err(SchemaError.fromZod(parsed.error, json, req));
      }
      const { current, success } = parsed.data;
      return Ok({
        current,
        reset: req.reset,
        pass: success,
        remaining: req.limit - current,
        triggered: success ? null : req.name,
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
