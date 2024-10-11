import { mysqlTable, varchar, text, timestamp, int, primaryKey, mysqlEnum, index, unique } from "drizzle-orm/mysql-core";
import { relations } from "drizzle-orm";
import { createInsertSchema, createSelectSchema } from "drizzle-zod";
import { keywords } from "./keywords";
import type { z } from "zod";
import { entries } from "./entries";

export const sections = mysqlTable("sections", {
  id: int("id").primaryKey().autoincrement(),
  entryId: int("entry_id").notNull(),
  heading: varchar("heading", { length: 255 }).notNull(),
  description: text("description").notNull(),
  order: int("order").notNull(),
});

export const sectionsRelations = relations(sections, ({ one, many }) => ({
  entry: one(entries, {
    fields: [sections.entryId],
    references: [entries.id],
  }),
  contentType: one(sectionContentTypes, {
    fields: [sections.id],
    references: [sectionContentTypes.sectionId],
  }),
  sectionsToKeywords: many(sectionsToKeywords),
}));

export const insertSectionSchema = createInsertSchema(sections)
  .extend({})
  .omit({ id: true });
export const selectSectionSchema = createSelectSchema(sections);
export type InsertSection = z.infer<typeof insertSectionSchema>;
export type SelectSection = typeof sections.$inferSelect;
const contentTypes = [
    'listicle',
    'table',
    'image',
    'code',
    'infographic',
    'timeline',
    'other'
] as const;

export const sectionContentTypes = mysqlTable("section_content_types", {
  id: int("id").primaryKey().autoincrement(),
  sectionId: int("section_id").notNull().unique(), // Ensures one-to-one relationship
  type: mysqlEnum('type', contentTypes).notNull(),
  description: text("description").notNull(),
  whyToUse: text("why_to_use").notNull(),
});

export const sectionContentTypesRelations = relations(sectionContentTypes, ({ one }) => ({
  section: one(sections, {
    fields: [sectionContentTypes.sectionId],
    references: [sections.id],
  }),
}));

export const insertSectionContentTypeSchema = createInsertSchema(sectionContentTypes)
  .extend({})
  .omit({ id: true });
export const selectSectionContentTypeSchema = createSelectSchema(sectionContentTypes);

export type InsertSectionContentType = z.infer<typeof insertSectionContentTypeSchema>;
export type SelectSectionContentType = typeof sectionContentTypes.$inferSelect;

export const sectionsToKeywords = mysqlTable(
  "sections_to_keywords",
  {
    sectionId: int("section_id").notNull(),
    keywordId: int("keyword_id").notNull(),
  },
  (t) => ({
    pk: primaryKey({ columns: [t.sectionId, t.keywordId] }),
  }),
);

export const selectSectionsToKeywordsSchema = createSelectSchema(sectionsToKeywords);
export const insertSectionsToKeywordsSchema = createInsertSchema(sectionsToKeywords)
    .extend({})

export type InsertSectionsToKeywords = z.infer<typeof insertSectionsToKeywordsSchema>;
export type SelectSectionsToKeywords = typeof sectionsToKeywords.$inferSelect;

export const sectionsToKeywordsRelations = relations(sectionsToKeywords, ({ one }) => ({
  section: one(sections, {
    fields: [sectionsToKeywords.sectionId],
    references: [sections.id],
  }),
  keyword: one(keywords, {
    fields: [sectionsToKeywords.keywordId],
    references: [keywords.id],
  }),
}));
