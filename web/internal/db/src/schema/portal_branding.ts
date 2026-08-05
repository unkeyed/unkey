import { relations } from "drizzle-orm";
import { bigint, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { portalConfigurations } from "./portal_configurations";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";

export const portalBranding = mysqlTable("portal_branding", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
  portalConfigId: id("portal_config_id").notNull().unique(),
  logoUrl: varchar("logo_url", { length: 500 }),
  primaryColor: varchar("primary_color", { length: 7 }),
  ...lifecycleDates,
});

export const portalBrandingRelations = relations(portalBranding, ({ one }) => ({
  portalConfiguration: one(portalConfigurations, {
    fields: [portalBranding.portalConfigId],
    references: [portalConfigurations.id],
  }),
}));
