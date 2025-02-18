import { relations } from "drizzle-orm";
import { index, int, json, mysqlTable, text, timestamp, varchar } from "drizzle-orm/mysql-core";
import { createSelectSchema } from "drizzle-zod";
import type { z } from "zod";
import { firecrawlResponses } from "./firecrawl";
import { searchQueries } from "./searchQuery";

// Main SearchResponse table
export const serperSearchResponses = mysqlTable(
  "serper_search_responses",
  {
    id: int("id").primaryKey().autoincrement(),
    inputTerm: varchar("input_term", { length: 255 }).notNull(),
    searchParameters: json("search_parameters").notNull(),
    answerBox: json("answer_box"),
    knowledgeGraph: json("knowledge_graph"),
    createdAt: timestamp("created_at").notNull().defaultNow(),
    updatedAt: timestamp("updated_at").notNull().defaultNow().onUpdateNow(),
  },
  (table) => ({
    inputTermIdx: index("input_term_idx").on(table.inputTerm),
  }),
);

export const serperSearchResponsesRelations = relations(serperSearchResponses, ({ one, many }) => ({
  searchQuery: one(searchQueries, {
    fields: [serperSearchResponses.inputTerm],
    references: [searchQueries.inputTerm],
  }),
  serperOrganicResults: many(serperOrganicResults),
  serperTopStories: many(serperTopStories),
  serperPeopleAlsoAsk: many(serperPeopleAlsoAsk),
  serperRelatedSearches: many(serperRelatedSearches),
}));

export const insertSearchResponseSchema = createSelectSchema(serperSearchResponses)
  .extend({})
  .omit({
    id: true,
  });

export type NewSearchResponseParams = z.infer<typeof insertSearchResponseSchema>;

export const serperOrganicResults = mysqlTable(
  "serper_organic_results",
  {
    id: int("id").primaryKey().autoincrement(),
    searchResponseId: int("search_response_id").notNull(),
    firecrawlResponseId: int("firecrawl_response_id"),
    title: varchar("title", { length: 255 }).notNull(),
    link: varchar("link", { length: 767 }).notNull(),
    snippet: text("snippet").notNull(),
    position: int("position").notNull(),
    imageUrl: varchar("image_url", { length: 767 }),
    createdAt: timestamp("created_at").notNull().defaultNow(),
    updatedAt: timestamp("updated_at").notNull().defaultNow().onUpdateNow(),
  },
  (table) => ({
    searchResponseIdIdx: index("search_response_id_idx").on(table.searchResponseId),
    linkIdx: index("link_idx").on(table.link),
  }),
);
export const serperOrganicResultsRelations = relations(serperOrganicResults, ({ one, many }) => ({
  searchResponse: one(serperSearchResponses, {
    fields: [serperOrganicResults.searchResponseId],
    references: [serperSearchResponses.id],
  }),
  sitelinks: many(serperSitelinks),
  firecrawlResponse: one(firecrawlResponses, {
    fields: [serperOrganicResults.link],
    references: [firecrawlResponses.sourceUrl],
  }),
}));

export const insertOrganicResultSchema = createSelectSchema(serperOrganicResults).extend({}).omit({
  id: true,
});

export type NewOrganicResultParams = z.infer<typeof insertOrganicResultSchema>;

export const serperSitelinks = mysqlTable(
  "serper_sitelinks",
  {
    id: int("id").primaryKey().autoincrement(),
    organicResultId: int("organic_result_id").notNull(),
    title: varchar("title", { length: 255 }).notNull(),
    link: varchar("link", { length: 767 }).notNull(),
    createdAt: timestamp("created_at").notNull().defaultNow(),
    updatedAt: timestamp("updated_at").notNull().defaultNow().onUpdateNow(),
  },
  (table) => ({
    organicResultIdIdx: index("organic_result_id_idx").on(table.organicResultId),
    linkIdx: index("link_idx").on(table.link),
  }),
);

export const serperSitelinksRelations = relations(serperSitelinks, ({ one }) => ({
  organicResult: one(serperOrganicResults, {
    fields: [serperSitelinks.organicResultId],
    references: [serperOrganicResults.id],
  }),
}));

// Schema for sitelinks - used to validate API requests
export const insertSitelinkSchema = createSelectSchema(serperSitelinks).extend({}).omit({
  id: true,
});

// Type for sitelinks - used to type API request params and within Components
export type NewSitelinkParams = z.infer<typeof insertSitelinkSchema>;

// Top Stories table
export const serperTopStories = mysqlTable(
  "serper_top_stories",
  {
    id: int("id").primaryKey().autoincrement(),
    searchResponseId: int("search_response_id").notNull(),
    title: varchar("title", { length: 255 }).notNull(),
    link: varchar("link", { length: 767 }).notNull(),
    source: varchar("source", { length: 255 }).notNull(),
    date: varchar("date", { length: 255 }).notNull(),
    imageUrl: varchar("image_url", { length: 767 }).notNull(),
    createdAt: timestamp("created_at").notNull().defaultNow(),
    updatedAt: timestamp("updated_at").notNull().defaultNow().onUpdateNow(),
  },
  (table) => ({
    searchResponseIdIdx: index("search_response_id_idx").on(table.searchResponseId),
    linkIdx: index("link_idx").on(table.link),
  }),
);
export const serperTopStoriesRelations = relations(serperTopStories, ({ one }) => ({
  searchResponse: one(serperSearchResponses, {
    fields: [serperTopStories.searchResponseId],
    references: [serperSearchResponses.id],
  }),
}));
// Schema for topStories - used to validate API requests
export const insertTopStorySchema = createSelectSchema(serperTopStories).extend({}).omit({
  id: true,
});

// Type for topStories - used to type API request params and within Components
export type NewTopStoryParams = z.infer<typeof insertTopStorySchema>;

// People Also Ask table
export const serperPeopleAlsoAsk = mysqlTable(
  "serper_people_also_ask",
  {
    id: int("id").primaryKey().autoincrement(),
    searchResponseId: int("search_response_id").notNull(),
    question: varchar("question", { length: 255 }).notNull(),
    snippet: text("snippet").notNull(),
    title: varchar("title", { length: 255 }).notNull(),
    link: varchar("link", { length: 767 }).notNull(),
    createdAt: timestamp("created_at").notNull().defaultNow(),
    updatedAt: timestamp("updated_at").notNull().defaultNow().onUpdateNow(),
  },
  (table) => ({
    searchResponseIdIdx: index("search_response_id_idx").on(table.searchResponseId),
    linkIdx: index("link_idx").on(table.link),
  }),
);

export const serperPeopleAlsoAskRelations = relations(serperPeopleAlsoAsk, ({ one }) => ({
  searchResponse: one(serperSearchResponses, {
    fields: [serperPeopleAlsoAsk.searchResponseId],
    references: [serperSearchResponses.id],
  }),
}));

// Schema for peopleAlsoAsk - used to validate API requests
export const insertPeopleAlsoAskSchema = createSelectSchema(serperPeopleAlsoAsk).extend({}).omit({
  id: true,
});

// Type for peopleAlsoAsk - used to type API request params and within Components
export type NewPeopleAlsoAskParams = z.infer<typeof insertPeopleAlsoAskSchema>;

// Related Searches table
export const serperRelatedSearches = mysqlTable(
  "serper_related_searches",
  {
    id: int("id").primaryKey().autoincrement(),
    searchResponseId: int("search_response_id").notNull(),
    query: varchar("query", { length: 255 }).notNull(),
    createdAt: timestamp("created_at").notNull().defaultNow(),
    updatedAt: timestamp("updated_at").notNull().defaultNow().onUpdateNow(),
  },
  (table) => ({
    searchResponseIdIdx: index("search_response_id_idx").on(table.searchResponseId),
    queryIdx: index("query_idx").on(table.query),
  }),
);

export const serperRelatedSearchesRelations = relations(serperRelatedSearches, ({ one }) => ({
  searchResponse: one(serperSearchResponses, {
    fields: [serperRelatedSearches.searchResponseId],
    references: [serperSearchResponses.id],
  }),
}));

// Schema for relatedSearches - used to validate API requests
export const insertRelatedSearchSchema = createSelectSchema(serperRelatedSearches).extend({}).omit({
  id: true,
});

// Type for relatedSearches - used to type API request params and within Components
export type NewRelatedSearchParams = z.infer<typeof insertRelatedSearchSchema>;
