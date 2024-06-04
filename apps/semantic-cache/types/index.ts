import type { z } from "zod";
import type { eventSchema } from "../src/analytics";

export type LLMResponse = {
  id: string;
  content: string;
};

export type InitialAnalyticsEvent = {
  time: number;
  model: string;
  stream: boolean;
  query: string;
  vector: number[];
};

export type AnalyticsEvent = z.infer<typeof eventSchema>;
