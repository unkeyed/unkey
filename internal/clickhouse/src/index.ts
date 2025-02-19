import { getActiveKeysPerDay, getActiveKeysPerHour, getActiveKeysPerMonth } from "./active_keys";
import { getBillableRatelimits, getBillableVerifications } from "./billing";
import { Client, type Inserter, Noop, type Querier } from "./client";
import { getLatestVerifications } from "./latest_verifications";
import {
  getDailyLogsTimeseries,
  getHourlyLogsTimeseries,
  getLogs,
  getMinutelyLogsTimeseries,
} from "./logs";
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

export type ClickHouseConfig =
  | {
      url?: string;
      insertUrl?: never;
      queryUrl?: never;
    }
  | {
      url?: never;
      insertUrl: string;
      queryUrl: string;
    };

export class ClickHouse {
  public readonly querier: Querier;
  public readonly inserter: Inserter;

  constructor(config: ClickHouseConfig) {
    if (config.url) {
      const client = new Client({ url: config.url });
      this.querier = client;
      this.inserter = client;
    } else if (config.queryUrl && config.insertUrl) {
      this.querier = new Client({ url: config.queryUrl });
      this.inserter = new Client({ url: config.insertUrl });
    } else {
      this.querier = new Noop();
      this.inserter = new Noop();
    }
  }

  static fromEnv(): ClickHouse {
    return new ClickHouse({ url: process.env.CLICKHOUSE_URL });
  }

  public get verifications() {
    return {
      insert: insertVerification(this.inserter),
      logs: getLatestVerifications(this.querier),
      perHour: getVerificationsPerHour(this.querier),
      perDay: getVerificationsPerDay(this.querier),
      perMonth: getVerificationsPerMonth(this.querier),
      latest: getLatestVerifications(this.querier),
    };
  }
  public get activeKeys() {
    return {
      perHour: getActiveKeysPerHour(this.querier),
      perDay: getActiveKeysPerDay(this.querier),
      perMonth: getActiveKeysPerMonth(this.querier),
    };
  }
  public get ratelimits() {
    return {
      insert: insertRatelimit(this.inserter),
      logs: getRatelimitLogs(this.querier),
      latest: getRatelimitLastUsed(this.querier),
      timeseries: {
        perMinute: getRatelimitsPerMinute(this.querier),
        perHour: getRatelimitsPerHour(this.querier),
        perDay: getRatelimitsPerDay(this.querier),
        perMonth: getRatelimitsPerMonth(this.querier),
      },
    };
  }
  public get billing() {
    return {
      billableVerifications: getBillableVerifications(this.querier),
      billableRatelimits: getBillableRatelimits(this.querier),
    };
  }
  public get api() {
    return {
      insert: insertApiRequest(this.inserter),
      logs: getLogs(this.querier),
      timeseries: {
        perMinute: getMinutelyLogsTimeseries(this.querier),
        perHour: getHourlyLogsTimeseries(this.querier),
        perDay: getDailyLogsTimeseries(this.querier),
      },
    };
  }
  public get business() {
    return {
      activeWorkspaces: getActiveWorkspacesPerMonth(this.querier),
    };
  }
  public get telemetry() {
    return {
      insert: insertSDKTelemetry(this.inserter),
    };
  }
}
