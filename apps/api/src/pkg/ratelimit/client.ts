import { Err, Ok, type Result, SchemaError } from "@unkey/error";
import type { Logger } from "@unkey/worker-logging";
import type { Context } from "hono";
import { z } from "zod";
import type { Metrics } from "../metrics";
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
  constructor(opts: {
    namespace: DurableObjectNamespace;

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
    });
    return res;
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
      });
    }

    const p = this.callDurableObject({
      identifier: req.identifier,
      objectName: id,
      window,
      reset,
      cost,
      limit: req.limit,
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
      });
    }
    current += cost;
    this.cache.set(id, current);

    return Ok({
      pass: true,
      current,
      reset,
    });
  }

  private getStub(name: string): DurableObjectStub {
    return this.namespace.get(this.namespace.idFromName(name));
  }

  private async callDurableObject(req: {
    identifier: string;
    objectName: string;
    window: number;
    reset: number;
    cost: number;
    limit: number;
  }): Promise<Result<RatelimitResponse, RatelimitError>> {
    try {
      const url = `https://${this.domain}/limit`;

      const call = () =>
        this.getStub(req.objectName).fetch(url, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            reset: req.reset,
            cost: req.cost,
            limit: req.limit,
          }),
        });

      // try twice
      const res = await call().catch(async (e) => {
        this.logger.warn("calling the ratelimit DO failed, retrying ...", {
          identifier: req.identifier,
          error: (e as Error).message,
        });

        return call();
      });

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
