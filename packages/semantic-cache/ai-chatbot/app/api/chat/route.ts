import { kv } from "@vercel/kv";
import { OpenAIStream, StreamingTextResponse } from "ai";
import OpenAI from "openai";

import { auth } from "@/auth";
import { nanoid } from "@/lib/utils";

export const runtime = "edge";

const openai = new OpenAI({
  apiKey: process.env.OPENAI_API_KEY,
  baseURL: "https://llmcache.unkey.workers.dev",
});

export async function POST(req: Request) {
  const json = await req.json();
  const { messages, previewToken } = json;

  if (previewToken) {
    openai.apiKey = previewToken;
  }

  const res = await openai.chat.completions.create({
    model: "gpt-4",
    messages,
    stream: true,
  });

  const stream = OpenAIStream(res);

  return new StreamingTextResponse(stream);
}
