import { z } from "zod";

/**
 * @description Schema for glossary entry takeaways
 * @warning This is a duplicate of apps/billing/src/lib/db-marketing/schemas/takeaways-schema.ts
 * @todo Extract this schema into a shared package to ensure consistency with the billing app
 * @see apps/billing/src/lib/db-marketing/schemas/takeaways-schema.ts for the source of truth
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
