import { ClickHouse } from "@unkey/clickhouse";
import { FetchError, wrap } from "@unkey/error";
import { z } from "zod";
import { ClickHouseProxyClient } from "./clickhouse-proxy";

export class Analytics {
  private readonly clickhouse: ClickHouse;
  private readonly proxyClient?: ClickHouseProxyClient;

  constructor(opts: {
    clickhouseUrl: string;
    clickhouseProxyUrl?: string;
    clickhouseProxyToken?: string;
  }) {
    // Keep ClickHouse client for queries/reads
    this.clickhouse = new ClickHouse({ url: opts.clickhouseUrl });
    // Use proxy client for inserts/writes if configured
    if (opts.clickhouseProxyUrl && opts.clickhouseProxyToken) {
      this.proxyClient = new ClickHouseProxyClient(
        opts.clickhouseProxyUrl,
        opts.clickhouseProxyToken,
      );
    }
  }

  public get insertSdkTelemetry() {
    return this.clickhouse.inserter.insert({
      table: "telemetry.raw_sdks_v1",
      schema: z.object({
        request_id: z.string(),
        time: z.number().int(),
        runtime: z.string(),
        platform: z.string(),
        versions: z.array(z.string()),
      }),
    });
  }

  public get insertRatelimit() {
    if (this.proxyClient) {
      return async (event: {
        request_id: string;
        time: number;
        workspace_id: string;
        namespace_id: string;
        identifier: string;
        passed: boolean;
      }) => {
        return await wrap(
          // biome-ignore lint/style/noNonNullAssertion: proxyClient existence verified above
          this.proxyClient!.insertRatelimits([event]),
          (err) =>
            new FetchError({
              message: err.message,
              retry: true,
            }),
        );
      };
    }
    return this.clickhouse.ratelimits.insert;
  }

  public get insertKeyVerification() {
    if (this.proxyClient) {
      return async (event: {
        request_id: string;
        time: number;
        workspace_id: string;
        key_space_id: string;
        key_id: string;
        region: string;
        outcome: string;
        identity_id?: string;
        tags?: string[];
      }) => {
        return await wrap(
          // biome-ignore lint/style/noNonNullAssertion: proxyClient existence verified above
          this.proxyClient!.insertVerifications([event]),
          (err) =>
            new FetchError({
              message: err.message,
              retry: true,
            }),
        );
      };
    }
    return this.clickhouse.verifications.insert;
  }

  public get insertApiRequest() {
    if (this.proxyClient) {
      return async (event: {
        workspace_id: string;
        request_id: string;
        time: number;
        host: string;
        method: string;
        path: string;
        request_headers: string[];
        request_body: string;
        response_status: number;
        response_headers: string[];
        response_body: string;
        error: string;
        service_latency: number;
        user_agent: string;
        ip_address: string;
        region: string;
      }) => {
        return await wrap(
          // biome-ignore lint/style/noNonNullAssertion: proxyClient existence verified above
          this.proxyClient!.insertApiRequests([event]),
          (err) =>
            new FetchError({
              message: err.message,
              retry: true,
            }),
        );
      };
    }
    return this.clickhouse.api.insert;
  }

  public get getVerificationsDaily() {
    return this.clickhouse.verifications.timeseries.perDay;
  }

  /**
   * Use this sparingly, mostly for quick iterations
   */
  public get internalQuerier() {
    return this.clickhouse.querier;
  }
}
