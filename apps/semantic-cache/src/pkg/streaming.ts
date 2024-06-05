import { OpenAIStream } from "ai";
import { streamSSE } from "hono/streaming";
import { nanoid } from "nanoid";
import type { OpenAI } from "openai";

import type { AnalyticsEvent, InitialAnalyticsEvent } from "@/pkg/analytics";
import type { Context } from "./hono/app";
import {
  OpenAIResponse,
  createCompletionChunk,
  extractWord,
  getEmbeddings,
  parseMessagesToString,
} from "./util";

const MATCH_THRESHOLD = 0.9;

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
  stream: ManagedStream,
  event: InitialAnalyticsEvent,
  vector?: number[],
) {
  const { analytics, cache } = c.get("services");
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
    await cache.completion.set(id, { id, content: contentStr });
    const writeTime = Date.now();
    console.info(`Cached with ID: ${id}, time: ${writeTime - time}ms`);
    if (vector) {
      await c.env.VECTORIZE_INDEX.insert([{ id, values: vector }]);
    }

    const finalEvent: AnalyticsEvent = {
      ...event,
      cache: true,
      query: "",
      requestId: id,
      latency: writeTime - time,
      // TODO: I don't know where this `tokens` was supposed to be
      tokens: 0, //tokens.length,
      response: contentStr,
      workspaceId: "test",
      gatewayId: "test",
    };
    try {
      const res = await analytics.ingestLogs(finalEvent);
      console.info("Logs persisted in Tinybird", res);
    } catch (err) {
      console.error("Error persisting logs in Tinybird:", err);
    }
    console.info("Data cached in KV with ID:", id);
  } else {
    console.info("Data discarded, did not end properly.");
  }
}

export async function handleStreamingRequest(
  c: Context,
  request: OpenAI.Chat.Completions.ChatCompletionCreateParamsStreaming & {
    noCache?: boolean;
  },
  openai: OpenAI,
) {
  const { cache } = c.get("services");
  c.header("Connection", "keep-alive");
  c.header("Cache-Control", "no-cache, must-revalidate");

  const startTime = Date.now();
  const messages = parseMessagesToString(request.messages);
  console.info("Messages:", messages);
  const vector = await getEmbeddings(c, messages);
  const embeddingsTime = performance.now();
  const query = await c.env.VECTORIZE_INDEX.query(vector, { topK: 1 });
  const queryTime = performance.now();

  const event = {
    time: Date.now(),
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
    c.executionCtx.waitUntil(handleCacheOrDiscard(c, managedStream, event, vector));

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
  const { val: data, err } = await cache.completion.get(query.matches[0].id);
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
    c.executionCtx.waitUntil(handleCacheOrDiscard(c, managedStream, event, vector));

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

export async function handleNonStreamingRequest(
  c: Context,
  request: OpenAI.Chat.Completions.ChatCompletionCreateParamsNonStreaming,
  openai: OpenAI,
) {
  const { cache } = c.get("services");
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
    await cache.completion.set(id, {
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
  const { val: data, err } = await cache.completion.get(query.matches[0].id);
  console.info(query.matches[0].id, data, err);
  const cacheFetchTime = Date.now();

  // If we have an embedding, we should always have a corresponding value in KV; but in case we don't,
  // regenerate and store it
  if (err || !data) {
    console.info("Vector identified, but no cached content found");
    const chatCompletion = await openai.chat.completions.create(request);
    await cache.completion.set(query.matches[0].id, {
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
