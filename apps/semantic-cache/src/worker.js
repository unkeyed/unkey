import { Hono } from "hono";
import { OpenAI } from "openai";
import { initCache } from "../lib/cache";
import { handleNonStreamingRequest, handleStreamingRequest } from "./index";
const app = new Hono();
app.post("/chat/completions", async (c) => {
  const openai = new OpenAI({
    apiKey: c.env.OPENAI_API_KEY,
  });
  const request = await c.req.json();
  const cache = await initCache(c);
  if (request.stream) {
    return handleStreamingRequest(c, request, openai, cache);
  }
  return handleNonStreamingRequest(c, request, openai, cache);
});
export default app;
