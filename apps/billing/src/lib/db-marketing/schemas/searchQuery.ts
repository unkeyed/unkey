import { relations, sql } from "drizzle-orm";
import {
  boolean,
  index,
  int,
  mysqlTable,
  timestamp,
  unique,
  varchar,
} from "drizzle-orm/mysql-core";
import { createInsertSchema } from "drizzle-zod";
import type { z } from "zod";
import { serperSearchResponses } from "./serper";

export const searchQueries = mysqlTable(
  "search_queries",
  {
    id: int("id").primaryKey().autoincrement(),
    inputTerm: varchar("input_term", { length: 255 }).notNull(),
    query: varchar("query", { length: 255 }).notNull(),
    isTermAsQueryAmbiguous: boolean("is_term_as_query_ambiguous").notNull().default(false),
    ambiguityReason: varchar("ambiguity_reason", { length: 255 }).notNull().default(""),
    clarifyingContext: varchar("clarifying_context", { length: 255 }).notNull().default(""),
    createdAt: timestamp("created_at").notNull().defaultNow(),
    updatedAt: timestamp("updated_at").notNull().defaultNow().onUpdateNow(),
  },
  (table) => ({
    inputTermIdx: index("input_term_idx").on(table.inputTerm),
    uniqueInputTerm: unique("search_queries_input_term_unique").on(table.inputTerm),
  }),
);

export const insertSearchQuerySchema = createInsertSchema(searchQueries).extend({});

export type NewSearchQueryParams = z.infer<typeof insertSearchQuerySchema>;

// every searchQuery can have an optional 1:1 serperResult searchResponses associated with it
// because the fk is stored in the serperResult table, the searchQueries relation have neither fields nor references
export const searchQueryRelations = relations(searchQueries, ({ one }) => ({
  searchResponses: one(serperSearchResponses, {
    fields: [searchQueries.inputTerm],
    references: [serperSearchResponses.inputTerm],
  }),
}));

export type SearchQuery = typeof searchQueries.$inferSelect;
export type NewSearchQuery = typeof searchQueries.$inferInsert;
