import { relations } from "drizzle-orm";
import {
  boolean,
  index,
  int,
  mysqlTable,
  text,
  timestamp,
  unique,
  varchar,
} from "drizzle-orm/mysql-core";
import { serperOrganicResults } from "./serper";

export const firecrawlResponses = mysqlTable(
  "firecrawl_responses",
  {
    id: int("id").primaryKey().autoincrement(),
    success: boolean("success").notNull(),
    scrapeId: text("scrape_id"),
    markdown: text("markdown"),
    sourceUrl: varchar("source_url", { length: 767 }).notNull(),
    statusCode: int("status_code"),
    title: varchar("title", { length: 767 }),
    description: text("description"),
    language: varchar("language", { length: 255 }),
    ogTitle: varchar("og_title", { length: 767 }),
    ogDescription: varchar("og_description", { length: 767 }),
    ogUrl: text("og_url"),
    ogImage: varchar("og_image", { length: 767 }),
    xogSiteName: varchar("og_site_name", { length: 767 }),
    createdAt: timestamp("created_at").defaultNow().notNull(),
    updatedAt: timestamp("updated_at").onUpdateNow(),
    error: text("error"),
  },
  (table) => ({
    sourceUrlIdx: index("source_url_idx").on(table.sourceUrl),
    uniqueSourceUrl: unique("unique_source_url").on(table.sourceUrl),
  }),
);

export const firecrawlResponsesRelations = relations(firecrawlResponses, ({ one }) => ({
  serperOrganicResult: one(serperOrganicResults, {
    fields: [firecrawlResponses.sourceUrl],
    references: [serperOrganicResults.link],
  }),
}));

export type FirecrawlResponse = typeof firecrawlResponses.$inferSelect;
export type NewFirecrawlResponse = typeof firecrawlResponses.$inferInsert;
