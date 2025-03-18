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
    inputTerm: varchar("input_term", { length: 767 }).notNull(),
    keyword: varchar("keyword", { length: 767 }).notNull(),
    inputTermAndKeywordHash: varchar("input_term_and_keyword_hash", { length: 64 }).notNull(), // so that we avoid having to halve the inputTerm/keywords length
    source: varchar("source", { length: 767 }).notNull(),
    sourceUrl: varchar("source_url", { length: 767 }),
    createdAt: timestamp("created_at").defaultNow().notNull(),
    updatedAt: timestamp("updated_at").defaultNow().notNull(),
  },
  (table) => ({
    inputTermIdx: index("input_term_idx").on(table.inputTerm),
    keywordIdx: index("keyword_idx").on(table.keyword),
    sourceUrlIdx: index("source_url_idx").on(table.sourceUrl),
    uniqueKeyword: unique("keywords_input_term_keyword_hash_unique").on(
      table.inputTermAndKeywordHash,
    ),
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
