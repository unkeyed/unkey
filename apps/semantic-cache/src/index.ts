import {
  CloudflareStore,
  DefaultStatefulContext,
  MemoryStore,
  Namespace,
  type Cache as UnkeyCache,
  createCache,
} from "@unkey/cache";

import { OpenAIStream } from "ai";

import { type Context, Hono } from "hono";
import { streamSSE } from "hono/streaming";
import { nanoid } from "nanoid";
import OpenAI from "openai";

import type { AnalyticsEvent, Bindings, InitialAnalyticsEvent, Response } from "../types";
import {
  OpenAIResponse,
  createCompletionChunk,
  extractWord,
  getEmbeddings,
  parseMessagesToString,
} from "../util";
import { Analytics } from "./analytics";

const MATCH_THRESHOLD = 0.9;

const app = new Hono<{ Bindings: Bindings }>();

class ManagedStream {
  stream: ReadableStream;
  reader: ReadableStreamDefaultReader<Uint8Array>;
  isDone: boolean;
  data: string;
  isComplete: boolean;

  constructor(stream: ReadableStream) {
    this.stream = stream;
    this.reader = this.stream.getReader();
    this.isDone = false;
    this.data = "";
    this.isComplete = false;
  }

  async readToEnd() {
    try {
      while (true) {
        const { done, value } = await this.reader.read();
        if (done) {
          this.isDone = true;
          break;
        }
        this.data += new TextDecoder().decode(value);
      }
    } catch (error) {
      console.error("Stream error:", error);
      this.isDone = false;
    } finally {
      this.reader.releaseLock();
    }
    return this.isDone;
  }

  checkComplete() {
    if (this.data.includes("[DONE]")) {
      this.isComplete = true;
    }
  }

  getReader() {
    return this.reader;
  }

  getData() {
    return this.data;
  }
}

async function handleCacheOrDiscard(
  c: Context,
  cache: UnkeyCache<{ response: Response }>,
  stream: ManagedStream,
  event: InitialAnalyticsEvent,
  vector?: number[],
) {
  await stream.readToEnd();

  // Check if the data is complete and should be cached
  if (stream.isDone) {
    const id = nanoid();
    const rawData = stream.getData();
    let contentStr = "";
    for (const token of rawData.split("\n")) {
      contentStr += extractWord(token);
    }
    const time = Date.now();
    await cache.response.set(id, { id, content: contentStr });
    const writeTime = Date.now();
    console.info(`Cached with ID: ${id}, time: ${writeTime - time}ms`);
    if (vector) {
      await c.env.VECTORIZE_INDEX.insert([{ id, values: vector }]);
    }

    const analytics = new Analytics({ tinybirdToken: c.env.TINYBIRD_TOKEN });
    const finalEvent = {
      ...event,
      cache: true,
      requestId: id,
      timing: writeTime - time,
      tokens: rawData.split("\n").length,
      response: contentStr,
    };
    analytics
      .ingestLogs(finalEvent)
      .then(() => {
        console.info("Logs persisted in Tinybird");
      })
      .catch((err) => {
        console.error("Error persisting logs in Tinybird:", err);
      });
    console.info("Data cached in KV with ID:", id);
  } else {
    console.info("Data discarded, did not end properly.");
  }
}

async function handleStreamingRequest(
  c: Context,
  request: OpenAI.Chat.Completions.ChatCompletionCreateParamsStreaming & {
    noCache?: boolean;
  },
  openai: OpenAI,
  cache: UnkeyCache<{ response: Response }>,
) {
  c.header("Connection", "keep-alive");
  c.header("Cache-Control", "no-cache, must-revalidate");

  const startTime = Date.now();
  const messages = parseMessagesToString(request.messages);
  console.info("Messages:", messages);
  const vector = await getEmbeddings(c, messages);
  const embeddingsTime = Date.now();
  const query = await c.env.VECTORIZE_INDEX.query(vector, { topK: 1 });
  const queryTime = Date.now();

  const event = {
    timestamp: new Date().toISOString(),
    model: request.model,
    stream: request.stream,
    query: messages as string,
    vector: [0],
  };

  console.info("Query results:", query);

  console.info(
    `Embeddings: ${embeddingsTime - startTime}ms, Query: ${queryTime - embeddingsTime}ms`,
  );

  // Cache miss
  if (query.count === 0 || query.matches[0].score < MATCH_THRESHOLD || request.noCache) {
    // strip no-cache from request
    const { noCache, ...requestOptions } = request;
    const chatCompletion = await openai.chat.completions.create(requestOptions);
    const responseStart = Date.now();
    console.info(`Response start: ${responseStart - queryTime}ms`);
    const stream = OpenAIStream(chatCompletion);
    const [stream1, stream2] = stream.tee();
    const managedStream = new ManagedStream(stream2);
    c.executionCtx.waitUntil(handleCacheOrDiscard(c, cache, managedStream, event, vector));

    return streamSSE(c, async (sseStream) => {
      const reader = stream1.getReader();
      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) {
            const responseEnd = Date.now();
            console.info(`Response end: ${responseEnd - responseStart}ms`);
            await sseStream.writeSSE({ data: "[DONE]" });
            break;
          }
          const data = new TextDecoder().decode(value);
          // extract token from SSE
          const formatted = extractWord(data);
          // format for OpenAI response
          const completionChunk = createCompletionChunk(formatted);
          // stringify
          const jsonString = JSON.stringify(completionChunk);
          // OpenAI have already formatted the string, so we need to unescape the newlines since Hono will do it again
          const correctedString = jsonString.replace(/\\\\n/g, "\\n");

          await sseStream.writeSSE({
            data: correctedString,
          });
        }
      } catch (error) {
        console.error("Stream error:", error);
      } finally {
        reader.releaseLock();
      }
    });
  }

  // Cache hit
  const { val: data, err } = await cache.response.get(query.matches[0].id);
  const cacheFetchTime = Date.now();

  console.info(`Cache fetch: ${cacheFetchTime - queryTime}ms`);

  // If we have an embedding, we should always have a corresponding value in KV; but in case we don't,
  // regenerate and store it
  if (!data || err) {
    // this repeats the logic above, except that we only write to the KV cache, not the vector DB
    const chatCompletion = await openai.chat.completions.create(request);
    const stream = OpenAIStream(chatCompletion);
    const [stream1, stream2] = stream.tee();
    const managedStream = new ManagedStream(stream2);
    c.executionCtx.waitUntil(handleCacheOrDiscard(c, cache, managedStream, event, vector));

    return streamSSE(c, async (sseStream) => {
      const reader = stream1.getReader();
      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) {
            await sseStream.writeSSE({ data: "[DONE]" });
            break;
          }
          const data = new TextDecoder().decode(value);
          const formatted = extractWord(data);
          await sseStream.writeSSE({
            data: JSON.stringify(createCompletionChunk(formatted)),
          });
        }
      } catch (error) {
        console.error("Stream error:", error);
      } finally {
        reader.releaseLock();
      }
    });
  }

  const wordsWithWhitespace = data.content.match(/\S+\s*/g) || "";

  return streamSSE(c, async (sseStream) => {
    for (const word of wordsWithWhitespace) {
      const completionChunk = createCompletionChunk(word);
      // stringify
      const jsonString = JSON.stringify(completionChunk);
      // OpenAI have already formatted the string, so we need to unescape the newlines since Hono will do it again
      const correctedString = jsonString.replace(/\\\\n/g, "\\n");

      await sseStream.writeSSE({
        data: correctedString,
      });
    }
    const endTime = Date.now();
    console.info(`SSE sending: ${endTime - cacheFetchTime}ms`);
  });
}

async function handleNonStreamingRequest(
  c: Context,
  request: OpenAI.Chat.Completions.ChatCompletionCreateParamsNonStreaming,
  openai: OpenAI,
  cache: UnkeyCache<{ response: Response }>,
) {
  const startTime = Date.now();
  const messages = parseMessagesToString(request.messages);

  const vector = await getEmbeddings(c, messages);
  const embeddingsTime = Date.now();
  const query = await c.env.VECTORIZE_INDEX.query(vector, { topK: 1 });
  const queryTime = Date.now();

  // Cache miss
  if (query.count === 0 || query.matches[0].score < MATCH_THRESHOLD) {
    const chatCompletion = await openai.chat.completions.create(request);
    const chatCompletionTime = Date.now();
    const id = nanoid();
    await c.env.VECTORIZE_INDEX.insert([{ id, values: vector }]);
    const vectorInsertTime = Date.now();
    await cache.response.set(id, {
      id,
      content: chatCompletion.choices[0].message.content || "",
    });
    const kvInsertTime = Date.now();
    console.info(
      `Embeddings: ${embeddingsTime - startTime}ms, Query: ${
        queryTime - embeddingsTime
      }ms, Chat Completion: ${chatCompletionTime - queryTime}ms, Vector Insert: ${
        vectorInsertTime - chatCompletionTime
      }ms, KV Insert: ${kvInsertTime - vectorInsertTime}ms`,
    );
    return c.json(chatCompletion);
  }

  // Cache hit
  const { val: data, err } = await cache.response.get(query.matches[0].id);
  console.info(query.matches[0].id, data, err);
  const cacheFetchTime = Date.now();

  // If we have an embedding, we should always have a corresponding value in KV; but in case we don't,
  // regenerate and store it
  if (err || !data) {
    console.info("Vector identified, but no cached content found");
    const chatCompletion = await openai.chat.completions.create(request);
    await cache.response.set(query.matches[0].id, {
      id: query.matches[0].id,
      content: chatCompletion.choices[0].message.content || "",
    });
    return c.json(chatCompletion);
  }

  console.info(
    `Embeddings: ${embeddingsTime - startTime}ms, Query: ${
      queryTime - embeddingsTime
    }ms, Cache Fetch: ${cacheFetchTime - queryTime}ms`,
  );
  return c.json(OpenAIResponse(data.content));
}

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

async function initCache(c: Context) {
  const context = new DefaultStatefulContext();
  const memory = new MemoryStore({
    persistentMap: new Map(),
  });
  const fresh = 6_000_000;
  const stale = 300_000_000;

  const cache = createCache({
    response: new Namespace<Response>(context, {
      stores: [
        memory,
        new CloudflareStore({
          cloudflareApiKey: c.env.CLOUDFLARE_API_KEY,
          zoneId: c.env.CLOUDFLARE_ZONE_ID,
          domain: "cache.unkey.dev",
        }),
      ],
      fresh,
      stale,
    }),
  });

  return cache;
}

export default app;
