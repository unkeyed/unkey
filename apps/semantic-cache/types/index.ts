import type { Ai } from "@cloudflare/ai";
import type { KVNamespace, VectorizeIndex } from "@cloudflare/workers-types";

export type LLMResponse = {
  id: string;
  content: string;
};

export type InitialAnalyticsEvent = {
  timestamp: string;
  model: string;
  stream: boolean;
  query: string;
  vector: number[];
};

export type AnalyticsEvent = InitialAnalyticsEvent & {
  response: string;
  cache: boolean;
  timing: number;
  tokens: number;
  requestId: string;
};

export type Bindings = {
  VECTORIZE_INDEX: VectorizeIndex;
  llmcache: KVNamespace;
  cache: any;
  OPENAI_API_KEY: string;
  TINYBIRD_TOKEN: string;
  AI: Ai;
};
