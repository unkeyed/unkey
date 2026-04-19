import { relations } from "drizzle-orm";
import { bigint, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { portalConfigurations } from "./portal_configurations";
import { lifecycleDates } from "./util/lifecycle_dates";

export const portalBranding = mysqlTable("portal_branding", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
  portalConfigId: varchar("portal_config_id", { length: 64 }).notNull().unique(),
  logoUrl: varchar("logo_url", { length: 500 }),
  primaryColor: varchar("primary_color", { length: 7 }),
  secondaryColor: varchar("secondary_color", { length: 7 }),
  ...lifecycleDates,
});

export const portalBrandingRelations = relations(portalBranding, ({ one }) => ({
  portalConfiguration: one(portalConfigurations, {
    fields: [portalBranding.portalConfigId],
    references: [portalConfigurations.id],
  }),
}));
