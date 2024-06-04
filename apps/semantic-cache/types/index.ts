import type { Ai } from "@cloudflare/ai";
import type { KVNamespace, VectorizeIndex } from "@cloudflare/workers-types";
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

export type Bindings = {
  VECTORIZE_INDEX: VectorizeIndex;
  llmcache: KVNamespace;
  cache: any;
  OPENAI_API_KEY: string;
  TINYBIRD_TOKEN: string;
  AI: Ai;
};
