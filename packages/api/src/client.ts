import type { PermissionQuery } from "@unkey/rbac";
import type { ErrorResponse } from "./errors";
import type { paths } from "./openapi";

import { type Telemetry, getTelemetry } from "./telemetry";

export type UnkeyOptions = (
  | {
      token?: never;

      /**
       * The root key from unkey.dev.
       *
       * You can create/manage your root keys here:
       * https://unkey.dev/app/settings/root-keys
       */
      rootKey: string;
    }
  | {
      /**
       * The workspace key from unkey.dev
       *
       * @deprecated Use `rootKey`
       */
      token: string;
      rootKey?: never;
    }
) & {
  /**
   * @default https://api.unkey.dev
   */
  baseUrl?: string;

  /**
   *
   * By default telemetry data is enabled, and sends:
   * runtime (Node.js / Edge)
   * platform (Node.js / Vercel / AWS)
   * SDK version
   */
  disableTelemetry?: boolean;

  /**
   * Retry on network errors
   */
  retry?: {
    /**
     * How many attempts should be made
     * The maximum number of requests will be `attempts + 1`
     * `0` means no retries
     *
     * @default 5
     */
    attempts?: number;
    /**
     * Return how many milliseconds to wait until the next attempt is made
     *
     * @default `(retryCount) => Math.round(Math.exp(retryCount) * 10)),`
     */
    backoff?: (retryCount: number) => number;
  };
  /**
   * Customize the `fetch` cache behaviour
   */
  cache?: RequestCache;

  /**
   * The version of the SDK instantiating this client.
   *
   * This is used for internal metrics and is not covered by semver, and may change at any time.
   *
   * You can leave this blank unless you are building a wrapper around this SDK.
   */
  wrapperSdkVersion?: string;
};

type ApiRequest = {
  path: string[];
} & (
  | {
      method: "GET";
      body?: never;
      query?: Record<string, string | number | boolean | null>;
    }
  | {
      method: "POST";
      body?: unknown;
      query?: never;
    }
);

type Result<R> =
  | {
      result: R;
      error?: never;
    }
  | {
      result?: never;
      error: ErrorResponse["error"];
    };

export class Unkey {
  public readonly baseUrl: string;
  private readonly rootKey: string;
  private readonly cache?: RequestCache;
  private readonly telemetry?: Telemetry | null;

  public readonly retry: {
    attempts: number;
    backoff: (retryCount: number) => number;
  };

  constructor(opts: UnkeyOptions) {
    this.baseUrl = opts.baseUrl ?? "https://api.unkey.dev";
    this.rootKey = opts.rootKey ?? opts.token;
    if (!opts.disableTelemetry) {
      this.telemetry = getTelemetry(opts);
    }

    this.cache = opts.cache;
    /**
     * Even though typescript should prevent this, some people still pass undefined or empty strings
     */
    if (!this.rootKey) {
      throw new Error(
        "Unkey root key must be set, maybe you passed in `undefined` or an empty string?",
      );
    }

    this.retry = {
      attempts: opts.retry?.attempts ?? 5,
      backoff: opts.retry?.backoff ?? ((n) => Math.round(Math.exp(n) * 10)),
    };
  }

  private getHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      Authorization: `Bearer ${this.rootKey}`,
    };
    if (this.telemetry?.sdkVersions) {
      headers["Unkey-Telemetry-SDK"] = this.telemetry.sdkVersions.join(",");
    }
    if (this.telemetry?.platform) {
      headers["Unkey-Telemetry-Platform"] = this.telemetry.platform;
    }
    if (this.telemetry?.runtime) {
      headers["Unkey-Telemetry-Runtime"] = this.telemetry.runtime;
    }
    return headers;
  }

  private async fetch<TResult>(req: ApiRequest): Promise<Result<TResult>> {
    let res: Response | null = null;
    let err: Error | null = null;
    for (let i = 0; i <= this.retry.attempts; i++) {
      const url = new URL(`${this.baseUrl}/${req.path.join("/")}`);
      if (req.query) {
        for (const [k, v] of Object.entries(req.query)) {
          if (v === null) {
            continue;
          }
          url.searchParams.set(k, v.toString());
        }
      }
      res = await fetch(url, {
        method: req.method,
        headers: this.getHeaders(),
        cache: this.cache,
        body: JSON.stringify(req.body),
      }).catch((e: Error) => {
        err = e;
        return null; // set `res` to `null`
      });
      if (res?.ok) {
        return { result: (await res.json()) as TResult };
      }
      const backoff = this.retry.backoff(i);
      console.debug(
        "attempt %d of %d to reach %s failed, retrying in %d ms: %s | %s",
        i + 1,
        this.retry.attempts + 1,
        url,
        backoff,
        // @ts-ignore I don't understand why `err` is `never`
        err?.message ?? `status=${res?.status}`,
        res?.headers.get("unkey-request-id"),
      );
      await new Promise((r) => setTimeout(r, backoff));
    }

    if (res) {
      return { error: (await res.json()) as ErrorResponse["error"] };
    }

    return {
      error: {
        // @ts-ignore
        code: "FETCH_ERROR",
        // @ts-ignore I don't understand why `err` is `never`
        message: err?.message ?? "No response",
        docs: "https://developer.mozilla.org/en-US/docs/Web/API/fetch",
        requestId: "N/A",
      },
    };
  }

  public get keys() {
    return {
      create: async (
        req: paths["/v1/keys.createKey"]["post"]["requestBody"]["content"]["application/json"],
      ): Promise<
        Result<
          paths["/v1/keys.createKey"]["post"]["responses"]["200"]["content"]["application/json"]
        >
      > => {
        return await this.fetch({
          path: ["v1", "keys.createKey"],
          method: "POST",
          body: req,
        });
      },
      update: async (
        req: paths["/v1/keys.updateKey"]["post"]["requestBody"]["content"]["application/json"],
      ): Promise<
        Result<
          paths["/v1/keys.updateKey"]["post"]["responses"]["200"]["content"]["application/json"]
        >
      > => {
        return await this.fetch({
          path: ["v1", "keys.updateKey"],
          method: "POST",
          body: req,
        });
      },
      verify: async <TPermission extends string = string>(
        req: Omit<
          paths["/v1/keys.verifyKey"]["post"]["requestBody"]["content"]["application/json"],
          "authorization"
        > & { authorization?: { permissions: PermissionQuery<TPermission> } },
      ): Promise<
        Result<
          paths["/v1/keys.verifyKey"]["post"]["responses"]["200"]["content"]["application/json"]
        >
      > => {
        return await this.fetch({
          path: ["v1", "keys.verifyKey"],
          method: "POST",
          body: req,
        });
      },
      delete: async (
        req: paths["/v1/keys.deleteKey"]["post"]["requestBody"]["content"]["application/json"],
      ): Promise<
        Result<
          paths["/v1/keys.deleteKey"]["post"]["responses"]["200"]["content"]["application/json"]
        >
      > => {
        return await this.fetch({
          path: ["v1", "keys.deleteKey"],
          method: "POST",
          body: req,
        });
      },
      updateRemaining: async (
        req: paths["/v1/keys.updateRemaining"]["post"]["requestBody"]["content"]["application/json"],
      ): Promise<
        Result<
          paths["/v1/keys.updateRemaining"]["post"]["responses"]["200"]["content"]["application/json"]
        >
      > => {
        return await this.fetch({
          path: ["v1", "keys.updateRemaining"],
          method: "POST",
          body: req,
        });
      },
      get: async (
        req: paths["/v1/keys.getKey"]["get"]["parameters"]["query"],
      ): Promise<
        Result<paths["/v1/keys.getKey"]["get"]["responses"]["200"]["content"]["application/json"]>
      > => {
        return await this.fetch({
          path: ["v1", "keys.getKey"],
          method: "GET",
          query: req,
        });
      },
      getVerifications: async (
        req: paths["/v1/keys.getVerifications"]["get"]["parameters"]["query"],
      ): Promise<
        Result<
          paths["/v1/keys.getVerifications"]["get"]["responses"]["200"]["content"]["application/json"]
        >
      > => {
        return await this.fetch({
          path: ["v1", "keys.getVerifications"],
          method: "GET",
          query: req,
        });
      },
    };
  }

  public get apis() {
    return {
      create: async (
        req: paths["/v1/apis.createApi"]["post"]["requestBody"]["content"]["application/json"],
      ): Promise<
        Result<
          paths["/v1/apis.createApi"]["post"]["responses"]["200"]["content"]["application/json"]
        >
      > => {
        return await this.fetch({
          path: ["v1", "apis.createApi"],
          method: "POST",
          body: req,
        });
      },
      delete: async (
        req: paths["/v1/apis.deleteApi"]["post"]["requestBody"]["content"]["application/json"],
      ): Promise<
        Result<
          paths["/v1/apis.deleteApi"]["post"]["responses"]["200"]["content"]["application/json"]
        >
      > => {
        return await this.fetch({
          path: ["v1", "apis.deleteApi"],
          method: "POST",
          body: req,
        });
      },
      get: async (
        req: paths["/v1/apis.getApi"]["get"]["parameters"]["query"],
      ): Promise<
        Result<paths["/v1/apis.getApi"]["get"]["responses"]["200"]["content"]["application/json"]>
      > => {
        return await this.fetch({
          path: ["v1", "apis.getApi"],
          method: "GET",
          query: req,
        });
      },
      listKeys: async (
        req: paths["/v1/apis.listKeys"]["get"]["parameters"]["query"],
      ): Promise<
        Result<paths["/v1/apis.listKeys"]["get"]["responses"]["200"]["content"]["application/json"]>
      > => {
        return await this.fetch({
          path: ["v1", "apis.listKeys"],
          method: "GET",
          query: req,
        });
      },
    };
  }
  public get ratelimits() {
    return {
      limit: async (
        req: paths["/v1/ratelimits.limit"]["post"]["requestBody"]["content"]["application/json"],
      ): Promise<
        Result<
          paths["/v1/ratelimits.limit"]["post"]["responses"]["200"]["content"]["application/json"]
        >
      > => {
        return await this.fetch({
          path: ["v1", "ratelimits.limit"],
          method: "POST",
          body: req,
        });
      },
    };
  }
}
