import { ClickHouse } from "@unkey/clickhouse";
import { z } from "zod";
import { ClickHouseProxyClient } from "./clickhouse-proxy";

export class Analytics {
  private readonly clickhouse: ClickHouse;
  private readonly proxyClient?: ClickHouseProxyClient;

  constructor(opts: {
    clickhouseUrl: string;
    clickhouseInsertUrl?: string;
    clickhouseProxyToken?: string;
  }) {
    // Keep ClickHouse client for queries/reads
    this.clickhouse = new ClickHouse({ url: opts.clickhouseUrl });

    // Use proxy client for inserts/writes if configured
    if (opts.clickhouseInsertUrl && opts.clickhouseProxyToken) {
      this.proxyClient = new ClickHouseProxyClient(
        opts.clickhouseInsertUrl,
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
        try {
          await this.proxyClient?.insertRatelimits([event]);
          return { err: null };
        } catch (err) {
          return { err: err instanceof Error ? err : new Error(String(err)) };
        }
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
        try {
          await this.proxyClient?.insertVerifications([event]);
          return { err: null };
        } catch (err) {
          return { err: err instanceof Error ? err : new Error(String(err)) };
        }
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
        country: string;
        city: string;
        colo: string;
        continent: string;
      }) => {
        try {
          await this.proxyClient?.insertApiRequests([event]);
          return { err: null };
        } catch (err) {
          return { err: err instanceof Error ? err : new Error(String(err)) };
        }
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
