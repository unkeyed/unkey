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
import { createInsertSchema } from "drizzle-zod";
import type { z } from "zod";
import { searchQueries } from "./searchQuery";
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
    title: text("title"),
    description: text("description"),
    language: text("language"),
    ogTitle: text("og_title"),
    ogDescription: text("og_description"),
    ogUrl: text("og_url"),
    ogImage: text("og_image"),
    ogSiteName: text("og_site_name"),
    createdAt: timestamp("created_at").defaultNow().notNull(),
    updatedAt: timestamp("updated_at").onUpdateNow(),
    error: text("error"),
    inputTerm: varchar("input_term", { length: 767 }),
    summary: text("summary"),
  },
  (table) => ({
    sourceUrlIdx: index("source_url_idx").on(table.sourceUrl),
    uniqueSourceUrl: unique("unique_source_url").on(table.sourceUrl),
    inputTermIdx: index("input_term_idx").on(table.inputTerm),
  }),
);

export const firecrawlResponsesRelations = relations(firecrawlResponses, ({ one }) => ({
  serperOrganicResult: one(serperOrganicResults, {
    fields: [firecrawlResponses.sourceUrl],
    references: [serperOrganicResults.link],
  }),
  searchQuery: one(searchQueries, {
    fields: [firecrawlResponses.inputTerm],
    references: [searchQueries.inputTerm],
  }),
}));

export const insertFirecrawlResponseSchema = createInsertSchema(firecrawlResponses)
  .extend({})
  .omit({ id: true });
export type NewFirecrawlResponse = z.infer<typeof insertFirecrawlResponseSchema>;
export type FirecrawlResponse = typeof firecrawlResponses.$inferSelect;
