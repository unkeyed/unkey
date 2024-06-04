import type { LLMResponse } from "types";

export type CacheNamespaces = {
  completion: LLMResponse;
};

export type CacheNamespace = keyof CacheNamespaces;
