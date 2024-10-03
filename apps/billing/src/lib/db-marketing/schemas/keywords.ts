import { relations } from "drizzle-orm";
import { index, int, mysqlTable, timestamp, unique, varchar } from "drizzle-orm/mysql-core";
import { searchQueries } from "./searchQuery";
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

export const keywordsRelations = relations(keywords, ({ one }) => ({
  inputTerm: one(searchQueries, {
    fields: [keywords.inputTerm],
    references: [searchQueries.inputTerm],
  }),
  sourceUrl: one(serperOrganicResults, {
    fields: [keywords.sourceUrl],
    references: [serperOrganicResults.link],
  }),
}));
