import { getBillableRatelimits, getBillableVerifications } from "./billing";
import { Client, type Inserter, Noop, type Querier } from "./client";
import {
  getDailyActiveKeysTimeseries,
  getFiveMinutelyActiveKeysTimeseries,
  getFourHourlyActiveKeysTimeseries,
  getHourlyActiveKeysTimeseries,
  getMinutelyActiveKeysTimeseries,
  getMonthlyActiveKeysTimeseries,
  getSixHourlyActiveKeysTimeseries,
  getThirtyMinutelyActiveKeysTimeseries,
  getThreeDayActiveKeysTimeseries,
  getTwelveHourlyActiveKeysTimeseries,
  getTwoHourlyActiveKeysTimeseries,
  getWeeklyActiveKeysTimeseries,
} from "./keys/active_keys";
import { getKeysOverviewLogs } from "./keys/keys";
import { getLatestVerifications } from "./latest_verifications";
import {
  getDailyLogsTimeseries,
  getFifteenMinuteLogsTimeseries,
  getFiveMinuteLogsTimeseries,
  getFourHourlyLogsTimeseries,
  getHourlyLogsTimeseries,
  getLogs,
  getMinutelyLogsTimeseries,
  getSixHourlyLogsTimeseries,
  getThirtyMinuteLogsTimeseries,
  getTwoHourlyLogsTimeseries,
} from "./logs";
import {
  getDailyLatencyTimeseries,
  getDailyRatelimitTimeseries,
  getFifteenMinuteLatencyTimeseries,
  getFifteenMinuteRatelimitTimeseries,
  getFiveMinuteLatencyTimeseries,
  getFiveMinuteRatelimitTimeseries,
  getFourHourlyLatencyTimeseries,
  getFourHourlyRatelimitTimeseries,
  getHourlyLatencyTimeseries,
  getHourlyRatelimitTimeseries,
  getMinutelyLatencyTimeseries,
  getMinutelyRatelimitTimeseries,
  getMonthlyRatelimitTimeseries,
  getRatelimitLastUsed,
  getRatelimitLogs,
  getRatelimitOverviewLogs,
  getSixHourlyLatencyTimeseries,
  getSixHourlyRatelimitTimeseries,
  getThirtyMinuteLatencyTimeseries,
  getThirtyMinuteRatelimitTimeseries,
  getTwoHourlyLatencyTimeseries,
  getTwoHourlyRatelimitTimeseries,
  insertRatelimit,
} from "./ratelimits";
import { insertApiRequest } from "./requests";
import { getActiveWorkspacesPerMonth } from "./success";
import { insertSDKTelemetry } from "./telemetry";
import {
  getDailyVerificationTimeseries,
  getFiveMinutelyVerificationTimeseries,
  getFourHourlyVerificationTimeseries,
  getHourlyVerificationTimeseries,
  getKeyDetailsLogs,
  getMinutelyVerificationTimeseries,
  getMonthlyVerificationTimeseries,
  getSixHourlyVerificationTimeseries,
  getThirtyMinutelyVerificationTimeseries,
  getThreeDayVerificationTimeseries,
  getTwelveHourlyVerificationTimeseries,
  getTwoHourlyVerificationTimeseries,
  getWeeklyVerificationTimeseries,
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
      latest: getLatestVerifications(this.querier),
      timeseries: {
        // Minute-based granularity
        perMinute: getMinutelyVerificationTimeseries(this.querier),
        per5Minutes: getFiveMinutelyVerificationTimeseries(this.querier),
        per30Minutes: getThirtyMinutelyVerificationTimeseries(this.querier),
        // Hour-based granularity
        perHour: getHourlyVerificationTimeseries(this.querier),
        per2Hours: getTwoHourlyVerificationTimeseries(this.querier),
        per4Hours: getFourHourlyVerificationTimeseries(this.querier),
        per6Hours: getSixHourlyVerificationTimeseries(this.querier),
        per12Hours: getTwelveHourlyVerificationTimeseries(this.querier),
        // Day-based granularity
        perDay: getDailyVerificationTimeseries(this.querier),
        per3Days: getThreeDayVerificationTimeseries(this.querier),
        perWeek: getWeeklyVerificationTimeseries(this.querier),
        // Month-based granularity
        perMonth: getMonthlyVerificationTimeseries(this.querier),
      },
      activeKeysTimeseries: {
        // Minute-based granularity
        perMinute: getMinutelyActiveKeysTimeseries(this.querier),
        per5Minutes: getFiveMinutelyActiveKeysTimeseries(this.querier),
        per30Minutes: getThirtyMinutelyActiveKeysTimeseries(this.querier),
        // Hour-based granularity
        perHour: getHourlyActiveKeysTimeseries(this.querier),
        per2Hours: getTwoHourlyActiveKeysTimeseries(this.querier),
        per4Hours: getFourHourlyActiveKeysTimeseries(this.querier),
        per6Hours: getSixHourlyActiveKeysTimeseries(this.querier),
        per12Hours: getTwelveHourlyActiveKeysTimeseries(this.querier),
        // Day-based granularity
        perDay: getDailyActiveKeysTimeseries(this.querier),
        per3Days: getThreeDayActiveKeysTimeseries(this.querier),
        perWeek: getWeeklyActiveKeysTimeseries(this.querier),
        // Month-based granularity
        perMonth: getMonthlyActiveKeysTimeseries(this.querier),
      },
    };
  }
  public get ratelimits() {
    return {
      insert: insertRatelimit(this.inserter),
      logs: getRatelimitLogs(this.querier),
      latest: getRatelimitLastUsed(this.querier),
      timeseries: {
        perMinute: getMinutelyRatelimitTimeseries(this.querier),
        per5Minutes: getFiveMinuteRatelimitTimeseries(this.querier),
        per15Minutes: getFifteenMinuteRatelimitTimeseries(this.querier),
        per30Minutes: getThirtyMinuteRatelimitTimeseries(this.querier),
        perHour: getHourlyRatelimitTimeseries(this.querier),
        per2Hours: getTwoHourlyRatelimitTimeseries(this.querier),
        per4Hours: getFourHourlyRatelimitTimeseries(this.querier),
        per6Hours: getSixHourlyRatelimitTimeseries(this.querier),
        perDay: getDailyRatelimitTimeseries(this.querier),
        perMonth: getMonthlyRatelimitTimeseries(this.querier),
        latency: {
          perMinute: getMinutelyLatencyTimeseries(this.querier),
          per5Minutes: getFiveMinuteLatencyTimeseries(this.querier),
          per15Minutes: getFifteenMinuteLatencyTimeseries(this.querier),
          per30Minutes: getThirtyMinuteLatencyTimeseries(this.querier),
          perHour: getHourlyLatencyTimeseries(this.querier),
          per2Hours: getTwoHourlyLatencyTimeseries(this.querier),
          per4Hours: getFourHourlyLatencyTimeseries(this.querier),
          per6Hours: getSixHourlyLatencyTimeseries(this.querier),
          perDay: getDailyLatencyTimeseries(this.querier),
        },
      },
      overview: {
        logs: getRatelimitOverviewLogs(this.querier),
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
      keys: {
        logs: getKeysOverviewLogs(this.querier),
      },
      key: {
        logs: getKeyDetailsLogs(this.querier),
      },
      timeseries: {
        perMinute: getMinutelyLogsTimeseries(this.querier),
        per5Minutes: getFiveMinuteLogsTimeseries(this.querier),
        per15Minutes: getFifteenMinuteLogsTimeseries(this.querier),
        per30Minutes: getThirtyMinuteLogsTimeseries(this.querier),
        perHour: getHourlyLogsTimeseries(this.querier),
        per2Hours: getTwoHourlyLogsTimeseries(this.querier),
        per4Hours: getFourHourlyLogsTimeseries(this.querier),
        per6Hours: getSixHourlyLogsTimeseries(this.querier),
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
