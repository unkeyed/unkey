import { relations } from "drizzle-orm";
import { mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { portalConfigurations } from "./portal_configurations";
import { legacyId } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";

export const portalBranding = mysqlTable("portal_branding", {
  pk: primaryKey(),
  portalConfigId: legacyId("portal_config_id").notNull().unique(),
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
