import { z } from "zod";

/**
 * @description Schema for glossary entry takeaways
 * @sourceOfTruth This is the source of truth for the takeaways schema as it's used for database storage
 * @todo Extract this schema into a shared package to avoid duplication with apps/www
 */
export const takeawaysSchema = z.object({
  tldr: z.string(),
  definitionAndStructure: z.array(
    z.object({
      key: z.string(),
      value: z.string(),
    }),
  ),
  historicalContext: z.array(
    z.object({
      key: z.string(),
      value: z.string(),
    }),
  ),
  usageInAPIs: z.object({
    tags: z.array(z.string()),
    description: z.string(),
  }),
  bestPractices: z.array(z.string()),
  recommendedReading: z.array(
    z.object({
      title: z.string(),
      url: z.string(),
    }),
  ),
  didYouKnow: z.string(),
});

export type Takeaways = z.infer<typeof takeawaysSchema>;
