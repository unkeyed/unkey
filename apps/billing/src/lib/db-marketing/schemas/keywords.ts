import { relations } from "drizzle-orm";
import { index, int, mysqlTable, timestamp, unique, varchar } from "drizzle-orm/mysql-core";
import { createInsertSchema, createSelectSchema } from "drizzle-zod";
import type { z } from "zod";
import { searchQueries } from "./searchQuery";
import { sectionsToKeywords } from "./sections";
import { serperOrganicResults } from "./serper";

export const keywords = mysqlTable(
  "keywords",
  {
    id: int("id").primaryKey().autoincrement(),
    inputTerm: varchar("input_term", { length: 255 }).notNull(),
    keyword: varchar("keyword", { length: 255 }).notNull(),
    source: varchar("source", { length: 255 }).notNull(),
    sourceUrl: varchar("source_url", { length: 767 }),
    createdAt: timestamp("created_at").defaultNow().notNull(),
    updatedAt: timestamp("updated_at").defaultNow().notNull(),
  },
  (table) => ({
    inputTermIdx: index("input_term_idx").on(table.inputTerm),
    sourceUrlIdx: index("source_url_idx").on(table.sourceUrl),
    uniqueKeyword: unique("keywords_input_term_keyword_unique").on(table.inputTerm, table.keyword),
  }),
);

export const insertKeywordsSchema = createInsertSchema(keywords).extend({}).omit({ id: true });
export const selectKeywordsSchema = createSelectSchema(keywords);
export type InsertKeywords = z.infer<typeof insertKeywordsSchema>;
export type SelectKeywords = typeof keywords.$inferSelect;

export const keywordsRelations = relations(keywords, ({ one, many }) => ({
  inputTerm: one(searchQueries, {
    fields: [keywords.inputTerm],
    references: [searchQueries.inputTerm],
  }),
  sourceUrl: one(serperOrganicResults, {
    fields: [keywords.sourceUrl],
    references: [serperOrganicResults.link],
  }),
  sectionsToKeywords: many(sectionsToKeywords),
}));
