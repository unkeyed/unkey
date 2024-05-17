import type { Ai } from "@cloudflare/ai";
import type { KVNamespace, VectorizeIndex } from "@cloudflare/workers-types";

export type Response = {
  id: string;
  content: string;
};

export type AnalyticsEvent = {
  timestamp: string;
  model: string;
  stream: boolean;
  query: string;
  vector: number[];
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
