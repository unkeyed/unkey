import { create, fromBinary, toBinary } from "@bufbuild/protobuf";
import { MySqlDialect, getTableConfig } from "drizzle-orm/mysql-core";
import { drizzle } from "drizzle-orm/mysql2";
import { createPool } from "mysql2";
import { describe, expect, expectTypeOf, it } from "vitest";
import { Audience, DestinationConfigSchema } from "../gen/notification/v1/config_pb";
import * as schema from "./index";
import {
  notificationDeliveries,
  notificationDestinations,
  notificationPreferences,
  notificationSubscriptionTargets,
  notifications,
} from "./notifications";

describe("notification idempotency", () => {
  it.each([
    {
      scope: "source event within a workspace and notification type",
      table: notifications,
      indexName: "notifications_source_event_idx",
      columns: [
        notifications.workspaceId,
        notifications.notificationType,
        notifications.sourceEventId,
      ],
    },
    {
      scope: "personal preference within a workspace, type, and channel",
      table: notificationPreferences,
      indexName: "notification_preferences_user_type_channel_idx",
      columns: [
        notificationPreferences.workspaceId,
        notificationPreferences.userId,
        notificationPreferences.notificationType,
        notificationPreferences.channel,
      ],
    },
    {
      scope: "delivery target across overlapping subscriptions and provider retries",
      table: notificationDeliveries,
      indexName: "notification_deliveries_target_idx",
      columns: [
        notificationDeliveries.notificationId,
        notificationDeliveries.channel,
        notificationDeliveries.targetKey,
      ],
    },
  ])("deduplicates $scope", ({ table, indexName, columns }) => {
    const index = getTableConfig(table).indexes.find((index) => index.config.name === indexName);

    expect(index?.config.unique).toBe(true);
    expect(index?.config.columns).toEqual(columns);
    for (const column of columns) {
      expect(column.notNull).toBe(true);
      if (column.dataType === "custom") {
        expect(column.getSQLType()).toContain("COLLATE utf8mb4_0900_as_cs");
      }
    }
  });
});

describe("notification targets", () => {
  it("stores stable protobuf audience values in a nullable integer column", () => {
    type Target = typeof notificationSubscriptionTargets.$inferSelect;
    expectTypeOf<Target["audience"]>().toEqualTypeOf<Audience | null>();
    expect(notificationSubscriptionTargets.audience.getSQLType()).toBe("int unsigned");
    expect(notificationSubscriptionTargets.audience.notNull).toBe(false);
    expect([Audience.UNSPECIFIED, Audience.WORKSPACE_ADMINS, Audience.BILLING_CONTACTS]).toEqual([
      0, 1, 2,
    ]);
  });

  it("requires exactly one recipient selector and email for personal recipients", () => {
    const dialect = new MySqlDialect();
    const checks = getTableConfig(notificationSubscriptionTargets).checks.map((check) => ({
      name: check.name,
      sql: dialect.sqlToQuery(check.value).sql,
    }));

    expect(checks).toEqual([
      {
        name: "notification_targets_exactly_one_recipient",
        sql: "(`notification_subscription_targets`.`audience` IS NOT NULL) + (`notification_subscription_targets`.`user_id` IS NOT NULL) + (`notification_subscription_targets`.`destination_id` IS NOT NULL) = 1",
      },
      {
        name: "notification_targets_audience_specified",
        sql: "`notification_subscription_targets`.`audience` IS NULL OR `notification_subscription_targets`.`audience` > 0",
      },
      {
        name: "notification_targets_channel_recipient",
        sql: "(`notification_subscription_targets`.`channel` = 'email' AND `notification_subscription_targets`.`destination_id` IS NULL) OR (`notification_subscription_targets`.`channel` <> 'email' AND `notification_subscription_targets`.`destination_id` IS NOT NULL)",
      },
    ]);
  });

  it("joins destinations by workspace and channel as well as ID", () => {
    const pool = createPool({});
    const database = drizzle(pool, { schema, mode: "planetscale" });

    try {
      const query = database.query.notificationSubscriptions
        .findMany({
          with: { targets: { with: { destination: true } } },
        })
        .toSQL();

      expect(query.sql).toContain(
        "`notificationSubscriptions_targets`.`workspace_id` = `notificationSubscriptions`.`workspace_id`",
      );
      expect(query.sql).toContain(
        "`notificationSubscriptions_targets_destination`.`workspace_id` = `notificationSubscriptions_targets`.`workspace_id`",
      );
      expect(query.sql).toContain(
        "`notificationSubscriptions_targets_destination`.`id` = `notificationSubscriptions_targets`.`destination_id`",
      );
      expect(query.sql).toContain(
        "`notificationSubscriptions_targets_destination`.`channel` = `notificationSubscriptions_targets`.`channel`",
      );
    } finally {
      pool.end();
    }
  });
});

describe("notification destination config", () => {
  it.each([
    {
      channel: "discord" as const,
      config: create(DestinationConfigSchema, {
        destination: { case: "discord", value: { encryptedWebhookUrl: "d" } },
      }),
      hex: "0a030a0164",
    },
    {
      channel: "slack" as const,
      config: create(DestinationConfigSchema, {
        destination: { case: "slack", value: { encryptedWebhookUrl: "s" } },
      }),
      hex: "12030a0173",
    },
    {
      channel: "webhook" as const,
      config: create(DestinationConfigSchema, {
        destination: {
          case: "webhook",
          value: {
            encryptedUrl: "w",
            headers: [{ name: "x", encryptedValue: "h" }],
            encryptedSigningSecret: "s",
          },
        },
      }),
      hex: "1a0e0a017712060a01781201681a0173",
    },
  ])("stores $channel config as protobuf bytes", ({ channel, config, hex }) => {
    const pool = createPool({});
    const database = drizzle(pool);

    try {
      const query = database
        .insert(notificationDestinations)
        .values({
          id: "dest_test",
          workspaceId: "ws_test",
          name: "Test destination",
          channel,
          config: Buffer.from(toBinary(DestinationConfigSchema, config)),
        })
        .toSQL();
      const stored = query.params.find(Buffer.isBuffer);

      expect(notificationDestinations.config.getSQLType()).toBe("longblob");
      expect(stored).toEqual(Buffer.from(hex, "hex"));
      expect(fromBinary(DestinationConfigSchema, Buffer.from(hex, "hex"))).toEqual(config);
      expect(query.sql).not.toContain("config_version");
      expect(query.sql).not.toContain("encrypted_config");
    } finally {
      pool.end();
    }
  });
});
