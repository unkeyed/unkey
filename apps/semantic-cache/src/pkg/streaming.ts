import { OpenAIStream } from "ai";
import { streamSSE } from "hono/streaming";
import type { OpenAI } from "openai";

import type { Context } from "./hono/app";
import {
  OpenAIResponse,
  createCompletionChunk,
  extractWord,
  getEmbeddings,
  parseMessagesToString,
} from "./util";

import { Tokenizer } from "@/pkg/tokens";
import { sha256 } from "@unkey/hash";

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

async function parseStream(stream: ReadableStream): Promise<string> {
  const ms = new ManagedStream(stream);
  await ms.readToEnd();

  // Check if the data is complete and should be cached
  if (!ms.isDone) {
    console.error("stream is not done yet, can't cache");
    return "";
  }
  const rawData = ms.getData();
  let contentStr = "";
  for (const token of rawData.split("\n")) {
    contentStr += extractWord(token);
  }
  return contentStr;
}

export async function handleStreamingRequest(
  c: Context,
  request: OpenAI.Chat.Completions.ChatCompletionCreateParamsStreaming & {
    noCache?: boolean;
  },
  openai: OpenAI,
): Promise<Response> {
  const { cache } = c.get("services");
  c.header("Connection", "keep-alive");
  c.header("Cache-Control", "no-cache, must-revalidate");

  const messages = parseMessagesToString(request.messages);
  const tokens = (await Tokenizer.init()).count(messages);
  c.set("tokens", tokens);

  const startEmbeddings = performance.now();
  const vector = await getEmbeddings(c, messages);
  c.set("embeddingsLatency", performance.now() - startEmbeddings);
  c.set("vector", vector);
  const startVectorize = performance.now();
  const query = await c.env.VECTORIZE_INDEX.query(vector, { topK: 1 });
  c.set("vectorizeLatency", performance.now() - startVectorize);
  c.set("query", messages);

  // Cache miss
  if (query.count === 0 || query.matches[0].score < MATCH_THRESHOLD || request.noCache) {
    // strip no-cache from request
    const { noCache, ...requestOptions } = request;
    const chatCompletion = await openai.chat.completions.create(requestOptions);
    const responseStart = Date.now();
    const stream = OpenAIStream(chatCompletion);
    const [stream1, stream2] = stream.tee();

    const content = await parseStream(stream2);
    const id = await sha256(content);

    c.executionCtx.waitUntil(cache.completion.set(id, { id, content }));

    if (vector) {
      c.executionCtx.waitUntil(c.env.VECTORIZE_INDEX.insert([{ id, values: vector }]));
    }

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
          const completionChunk = await createCompletionChunk(formatted);
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

  c.set("cacheHit", true);

  // Cache hit
  const cacheStart = performance.now();
  const { val: cached, err } = await cache.completion.get(query.matches[0].id);
  c.set("cacheLatency", performance.now() - cacheStart);
  const cacheFetchTime = Date.now();

  // If we have an embedding, we should always have a corresponding value in the cache; but in case we don't,
  // regenerate and store it
  const inferenceStart = performance.now();
  if (!cached || err) {
    // this repeats the logic above, except that we only write to the KV cache, not the vector DB
    const chatCompletion = await openai.chat.completions.create(request);
    const stream = OpenAIStream(chatCompletion);
    const [stream1, stream2] = stream.tee();

    const content = await parseStream(stream2);
    const id = await sha256(content);

    c.executionCtx.waitUntil(cache.completion.set(id, { id, content }));

    if (vector) {
      c.executionCtx.waitUntil(c.env.VECTORIZE_INDEX.insert([{ id, values: vector }]));
    }

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
            data: JSON.stringify(await createCompletionChunk(formatted)),
          });
        }
      } catch (error) {
        console.error("Stream error:", error);
      } finally {
        reader.releaseLock();
        c.set("inferenceLatency", performance.now() - inferenceStart);
      }
    });
  }

  const wordsWithWhitespace = cached.content.match(/\S+\s*/g) || "";

  return streamSSE(c, async (sseStream) => {
    for (const word of wordsWithWhitespace) {
      const completionChunk = await createCompletionChunk(word);
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
    c.set("inferenceLatency", performance.now() - inferenceStart);
  });
}

export async function handleNonStreamingRequest(
  c: Context,
  request: OpenAI.Chat.Completions.ChatCompletionCreateParamsNonStreaming,
  openai: OpenAI,
): Promise<Response> {
  const { cache } = c.get("services");
  const messages = parseMessagesToString(request.messages);
  c.set("query", messages);
  const tokens = (await Tokenizer.init()).count(messages);
  c.set("tokens", tokens);

  const startEmbeddings = performance.now();
  const vector = await getEmbeddings(c, messages);
  c.set("embeddingsLatency", performance.now() - startEmbeddings);
  c.set("vector", vector);
  const startVectorize = performance.now();
  const query = await c.env.VECTORIZE_INDEX.query(vector, { topK: 1 });
  c.set("vectorizeLatency", performance.now() - startVectorize);

  // Cache miss
  if (query.count === 0 || query.matches[0].score < MATCH_THRESHOLD) {
    const inferenceStart = performance.now();
    const chatCompletion = await openai.chat.completions.create(request);
    c.set("inferenceLatency", performance.now() - inferenceStart);
    const content = chatCompletion.choices.at(0)?.message.content || "";
    const id = await sha256(content);

    c.executionCtx.waitUntil(cache.completion.set(id, { id, content }));

    if (vector) {
      c.executionCtx.waitUntil(c.env.VECTORIZE_INDEX.insert([{ id, values: vector }]));
    }

    return c.json(chatCompletion);
  }

  // Cache hit
  const cacheStart = performance.now();
  const { val: cached, err } = await cache.completion.get(query.matches[0].id);
  c.set("cacheLatency", performance.now() - cacheStart);
  // If we have an embedding, we should always have a corresponding value in the cache; but in case we don't,
  // regenerate and store it
  if (err || !cached) {
    console.info("Vector identified, but no cached content found");
    const chatCompletion = await openai.chat.completions.create(request);
    const id = query.matches[0].id;
    const content = chatCompletion.choices[0].message.content || "";
    await cache.completion.set(id, { id, content });
    return c.json(chatCompletion);
  }
  c.set("cacheHit", true);

  return c.json(OpenAIResponse(cached.content));
}
