import { index, int, mysqlTable, timestamp, varchar } from "drizzle-orm/mysql-core";
import { searchQueries } from "./searchQuery";
import { relations } from "drizzle-orm";
import { createInsertSchema, createSelectSchema } from "drizzle-zod";
import type { z } from "zod";
import { sections } from "./sections";

export const entries = mysqlTable(
  "entries",
  {
    id: int("id").primaryKey().autoincrement(),
    inputTerm: varchar("input_term", { length: 255 }).notNull(),
    utKey: varchar("ut_key", { length: 255 }),
    utUrl: varchar("ut_url", { length: 255 }),
    prUrl: varchar("pr_url", { length: 255 }),
    createdAt: timestamp("created_at").notNull().defaultNow(),
    updatedAt: timestamp("updated_at")
      .notNull()
      .defaultNow()
      .$onUpdate(() => new Date()),
  },
  (table) => ({
    inputTermIdx: index("input_term_idx").on(table.inputTerm),
  }),
);

export const entriesRelations = relations(entries, ({ many, one }) => ({
  dynamicSections: many(sections),
  searchQuery: one(searchQueries, {
    fields: [entries.inputTerm],
    references: [searchQueries.inputTerm],
  }),
}));

export const insertEntrySchema = createInsertSchema(entries).extend({}).omit({ id: true });
export const selectEntrySchema = createSelectSchema(entries);

export type InsertEntry = z.infer<typeof insertEntrySchema>;
export type SelectEntry = typeof entries.$inferSelect;
