import type { FieldConfig } from "@/components/logs/validation/filter.types";
import type { z } from "zod";

export type FilterGroup = {
  field: string;
  filters: { operator: string; value: string | number }[];
};

export type FilterOutput = { filters: FilterGroup[] };

export type OutputSchema = z.ZodType<FilterOutput>;

export type Example = {
  query: string;
  result: FilterGroup[];
  /** Why this answer, when the query alone does not make it obvious. */
  note?: string;
};

export type SearchSpec = {
  subject: string;
  outputSchema: OutputSchema;
  config: Record<string, FieldConfig>;
  /** Plain language per field, so the model knows what a value means rather than guessing. */
  fields: Record<string, string>;
  examples: Example[];
  /** Constraints this surface adds on top of the shared ones, one line each. */
  rules?: string[];
  /** How to break a tie this surface gets wrong without being told. */
  priorities?: string[];
  /** Carries a relative duration such as 24h, and unlocks the shared time rules. */
  durationField?: string;
};
