import { Unkey } from "@unkey/api";
import { version } from "../package.json";
import { type Duration, ms } from "./duration";
import type { Ratelimiter } from "./interface";
import type { Limit, LimitOptions, RatelimitResponse } from "./types";

export type RatelimitConfig = Limit & {
  /**
   * @default https://api.unkey.dev
   */
  baseUrl?: string;

  /**
   * The unkey root key. You can create one at https://unkey.dev/app/settings/root-keys
   *
   * Make sure the root key has permissions to use ratelimiting.
   */
  rootKey: string;

  /**
   * Namespaces allow you to separate different areas of your app and have isolated limits.
   *
   * @example tRPC-routes
   */
  namespace: string;

  /**
   * Configure a timeout to prevent network issues from blocking your function for too long.
   *
   * Disable it by setting `timeout: false`
   *
   * @default
   * ```ts
   * {
   *   // 5 seconds
   *   ms: 5000,
   *   fallback: { success: false, limit: 0, remaining: 0, reset: Date.now()}
   * }
   * ```
   */
  timeout?:
    | {
        /**
         * Time in milliseconds until the response is returned
         */
        ms: number | Duration;

        /**
         * A custom response to return when the timeout is reached.
         *
         * The important bit is the `success` value, choose whether you want to let requests pass or not.
         *
         * @example
         * ```ts
         * {
         *   // 5 seconds
         *   ms: 5000
         *   fallback: { success: true, limit: 0, remaining: 0, reset: 0}
         * }
         * ```
         */
        fallback: RatelimitResponse;
      }
    | false;

  /**
   * Configure what happens for unforeseen errors
   *
   * @example Letting requests pass
   * ```ts
   *   onError: ()=> ({ success: true, limit: 0, remaining: 0, reset: 0})
   * ```
   *
   * @example Rejecting the request
   * ```ts
   *   onError: ()=> ({ success: true, limit: 0, remaining: 0, reset: 0})
   * ```
   */
  onError?: (err: Error) => RatelimitResponse | Promise<RatelimitResponse>;

  /**
   * Do not wait for a response from the origin. Faster but less accurate.
   */
  async?: boolean;

  /**
   *
   * By default telemetry data is enabled, and sends:
   * runtime (Node.js / Edge)
   * platform (Node.js / Vercel / AWS)
   * SDK version
   */
  disableTelemetry?: boolean;
};

export class Ratelimit implements Ratelimiter {
  private readonly config: RatelimitConfig;
  private readonly unkey: Unkey;

  constructor(config: RatelimitConfig) {
    this.config = config;
    this.unkey = new Unkey({
      baseUrl: config.baseUrl,
      rootKey: config.rootKey,
      disableTelemetry: config.disableTelemetry,
      wrapperSdkVersion: `@unkey/ratelimit@${version}`,
    });
  }

  /**
   * Limit a specific identifier, you can override a lot of things about this specific request using
   * the 2nd argument.
   *
   * @example
   * ```ts
   * const identifier = getIpAddress() // or userId or anything else
   * const res = await unkey.limit(identifier)
   *
   * if (!res.success){
   *   // reject request
   * }
   * // handle request
   * ```
   */
  public async limit(identifier: string, opts?: LimitOptions): Promise<RatelimitResponse> {
    try {
      return this._limit(identifier, opts);
    } catch (e) {
      if (!this.config.onError) {
        throw e;
      }
      const err = e instanceof Error ? e : new Error(String(e));

      return await this.config.onError(err);
    }
  }
  private async _limit(identifier: string, opts?: LimitOptions): Promise<RatelimitResponse> {
    const timeout =
      this.config.timeout === false
        ? null
        : this.config.timeout ?? {
            ms: 5000,
            fallback: { success: false, limit: 0, remaining: 0, reset: Date.now() },
          };

    let timeoutId: any = null;
    try {
      const ps: Promise<RatelimitResponse>[] = [
        this.unkey.ratelimits
          .limit({
            namespace: this.config.namespace,
            identifier,
            limit: this.config.limit,
            duration: ms(this.config.duration),
            cost: opts?.cost,
            meta: opts?.meta,
            resources: opts?.resources,
            async: typeof opts?.async !== "undefined" ? opts.async : this.config.async,
          })
          .then((res) => {
            if (res.error) {
              throw new Error(
                `Ratelimit failed: [${res.error.code} - ${res.error.requestId}]: ${res.error.message}`,
              );
            }
            return res.result;
          }),
      ];
      if (timeout) {
        ps.push(
          new Promise((resolve) => {
            timeoutId = setTimeout(() => {
              resolve(timeout.fallback);
            }, ms(timeout.ms));
          }),
        );
      }

      return await Promise.race(ps);
    } finally {
      if (timeoutId) {
        clearTimeout(timeoutId);
      }
    }
  }
}
