import { getActiveKeysPerDay, getActiveKeysPerHour, getActiveKeysPerMonth } from "./active_keys";
import { getBillableRatelimits, getBillableVerifications } from "./billing";
import { Client, type Inserter, Noop, type Querier } from "./client";
import { getLatestVerifications } from "./latest_verifications";
import { getLogs } from "./logs";
import {
  getRatelimitLastUsed,
  getRatelimitLogs,
  getRatelimitsPerDay,
  getRatelimitsPerHour,
  getRatelimitsPerMinute,
  getRatelimitsPerMonth,
  insertRatelimit,
} from "./ratelimits";
import { insertApiRequest } from "./requests";
import { getActiveWorkspacesPerMonth } from "./success";
import { insertSDKTelemetry } from "./telemetry";
import {
  getVerificationsPerDay,
  getVerificationsPerHour,
  getVerificationsPerMonth,
  insertVerification,
} from "./verifications";

export type ClickHouseConfig = {
  url?: string;
};

export class ClickHouse {
  public readonly client: Querier & Inserter;

  constructor(config: ClickHouseConfig) {
    if (config.url) {
      this.client = new Client({ url: config.url });
    } else {
      this.client = new Noop();
    }
  }

  static fromEnv(): ClickHouse {
    return new ClickHouse({ url: process.env.CLICKHOUSE_URL });
  }
  public get verifications() {
    return {
      insert: insertVerification(this.client),
      logs: getLatestVerifications(this.client),
      perHour: getVerificationsPerHour(this.client),
      perDay: getVerificationsPerDay(this.client),
      perMonth: getVerificationsPerMonth(this.client),
      latest: getLatestVerifications(this.client),
    };
  }
  public get activeKeys() {
    return {
      perHour: getActiveKeysPerHour(this.client),
      perDay: getActiveKeysPerDay(this.client),
      perMonth: getActiveKeysPerMonth(this.client),
    };
  }
  public get ratelimits() {
    return {
      insert: insertRatelimit(this.client),
      logs: getRatelimitLogs(this.client),
      latest: getRatelimitLastUsed(this.client),
      perMinute: getRatelimitsPerMinute(this.client),
      perHour: getRatelimitsPerHour(this.client),
      perDay: getRatelimitsPerDay(this.client),
      perMonth: getRatelimitsPerMonth(this.client),
    };
  }
  public get billing() {
    return {
      billableVerifications: getBillableVerifications(this.client),
      billableRatelimits: getBillableRatelimits(this.client),
    };
  }
  public get api() {
    return {
      insert: insertApiRequest(this.client),
      logs: getLogs(this.client),
    };
  }
  public get business() {
    return {
      activeWorkspaces: getActiveWorkspacesPerMonth(this.client),
    };
  }
  public get telemetry() {
    return {
      insert: insertSDKTelemetry(this.client),
    };
  }
}
