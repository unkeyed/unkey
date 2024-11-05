import { getActiveKeysPerDay, getActiveKeysPerHour, getActiveKeysPerMonth } from "./active_keys";
import { Noop, type Inserter, type Querier, Client } from "./client";

export type ClickHouseConfig = {
  url?: string;
};

export class ClickHouse {
  private readonly client: Querier & Inserter;

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

  public get activeKeys() {
    return {
      perHour: getActiveKeysPerHour(this.client),
      perDay: getActiveKeysPerDay(this.client),
      perMonth: getActiveKeysPerMonth(this.client),
    };
  }
}
