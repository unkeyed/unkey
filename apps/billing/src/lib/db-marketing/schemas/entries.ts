import { relations } from "drizzle-orm";
import {
  index,
  int,
  json,
  mysqlEnum,
  mysqlTable,
  text,
  timestamp,
  varchar,
} from "drizzle-orm/mysql-core";
import { createInsertSchema, createSelectSchema } from "drizzle-zod";
import { z } from "zod";
import { searchQueries } from "./searchQuery";
import { sections } from "./sections";
import type { Takeaways } from "./takeaways-schema";

export const entryStatus = ["ARCHIVED", "PUBLISHED"] as const;
export type EntryStatus = (typeof entryStatus)[number];
export const faqSchema = z.array(
  z.object({
    question: z.string(),
    answer: z.string(),
  }),
);

export type FAQ = z.infer<typeof faqSchema>;

export const entries = mysqlTable(
  "entries",
  {
    id: int("id").primaryKey().autoincrement(),
    inputTerm: varchar("input_term", { length: 255 }).notNull(),
    githubPrUrl: varchar("github_pr_url", { length: 255 }),
    dynamicSectionsContent: text("dynamic_sections_content"),
    metaTitle: varchar("meta_title", { length: 255 }),
    metaDescription: varchar("meta_description", { length: 255 }),
    metaH1: varchar("meta_h1", { length: 255 }),
    categories: json("linking_categories").$type<string[]>().default([]),
    status: mysqlEnum("status", entryStatus),
    takeaways: json("content_takeaways").$type<Takeaways>(),
    faq: json("content_faq").$type<FAQ>(),
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
