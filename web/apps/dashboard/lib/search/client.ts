import { env } from "@/lib/env";
import OpenAI from "openai";

export const SEARCH_MODEL = "gpt-4o-mini";
export const SEARCH_TIMEOUT_MS = 30_000;

let cached: OpenAI | null | undefined;

export function searchClient(): OpenAI | null {
  if (cached === undefined) {
    const apiKey = env().OPENAI_API_KEY;
    cached = apiKey ? new OpenAI({ apiKey, timeout: SEARCH_TIMEOUT_MS }) : null;
  }
  return cached;
}
