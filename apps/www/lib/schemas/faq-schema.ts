import { z } from "zod";

/**
 * @description Schema for glossary entry FAQs
 * @warning This is a duplicate of apps/billing/src/lib/db-marketing/schemas/entries.ts
 * @todo Extract this schema into a shared package to ensure consistency with the billing app
 * @see apps/billing/src/lib/db-marketing/schemas/entries.ts for the source of truth
 */
export const faqSchema = z.array(
  z.object({
    question: z.string(),
    answer: z.string(),
  }),
);

export type FAQ = z.infer<typeof faqSchema>;
