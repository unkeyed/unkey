import { OpenAIStream } from "ai";
import { streamSSE } from "hono/streaming";
import type { OpenAI } from "openai";

import type { Context } from "./hono/app";
import { OpenAIResponse, createCompletionChunk, extractWord, parseMessagesToString } from "./util";

import { Tokenizer } from "@/pkg/tokens";
import type { CacheError } from "@unkey/cache";
import { BaseError, Err, Ok, type Result, wrap } from "@unkey/error";
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
  c.header("Connection", "keep-alive");
  c.header("Cache-Control", "no-cache, must-revalidate");

  const query = parseMessagesToString(request.messages);
  c.set("query", query);
  const tokens = (await Tokenizer.init()).count(query);
  c.set("tokens", tokens);

  const embeddings = await createEmbeddings(c, query);
  if (embeddings.err) {
    // TODO: handle error
    throw new Error(embeddings.err.message);
  }

  const cached = await loadCache(c, embeddings.val);
  if (cached.err) {
    // TODO: handle error
    throw new Error(cached.err.message);
  }
  // Cache hit
  if (cached.val) {
    const wordsWithWhitespace = cached.val.match(/\S+\s*/g) || "";

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
    });
  }

  // strip no-cache from request
  const { noCache, ...requestOptions } = request;
  const chatCompletion = await openai.chat.completions.create(requestOptions);
  const responseStart = Date.now();
  const stream = OpenAIStream(chatCompletion);
  const [stream1, stream2] = stream.tee();

  c.executionCtx.waitUntil(updateCache(c, embeddings.val, await parseStream(stream2)));

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

export async function handleNonStreamingRequest(
  c: Context,
  request: OpenAI.Chat.Completions.ChatCompletionCreateParamsNonStreaming,
  openai: OpenAI,
): Promise<Response> {
  const { logger } = c.get("services");
  const query = parseMessagesToString(request.messages);
  c.set("query", query);
  const tokens = (await Tokenizer.init()).count(query);
  c.set("tokens", tokens);

  const embeddings = await createEmbeddings(c, query);
  if (embeddings.err) {
    // TODO: handle error
    throw new Error(embeddings.err.message);
  }

  const cached = await loadCache(c, embeddings.val);
  if (cached.err) {
    // TODO: handle error
    throw new Error(cached.err.message);
  }
  // Cache hit
  if (cached.val) {
    return c.json(OpenAIResponse(cached.val));
  }

  // miss

  const inferenceStart = performance.now();
  const chatCompletion = await openai.chat.completions.create(request);
  c.set("inferenceLatency", performance.now() - inferenceStart);

  const { err: updateCacheError } = await updateCache(
    c,
    embeddings.val,
    chatCompletion.choices.at(0)?.message.content || "",
  );
  if (updateCacheError) {
    logger.error("unable to update cache", {
      error: updateCacheError.message,
    });
  }

  c.set("response", JSON.stringify(chatCompletion));
  return c.json(chatCompletion);
}

async function createEmbeddings(
  c: Context,
  text: string,
): Promise<Result<AiTextEmbeddingsOutput, CloudflareAiError>> {
  const startEmbeddings = performance.now();
  const embeddings = await wrap(
    c.env.AI.run("@cf/baai/bge-small-en-v1.5", {
      text,
    }),
    (err) => new CloudflareAiError({ message: err.message }),
  );
  c.set("embeddingsLatency", performance.now() - startEmbeddings);

  if (embeddings.err) {
    return Err(embeddings.err);
  }
  c.set("vector", embeddings.val.data[0]);
  return Ok(embeddings.val);
}

export class CloudflareAiError extends BaseError {
  public readonly retry = true;
  public readonly name = CloudflareAiError.name;
}

export class CloudflareVectorizeError extends BaseError {
  public readonly retry = true;
  public readonly name = CloudflareVectorizeError.name;
}

async function loadCache(
  c: Context,
  embeddings: AiTextEmbeddingsOutput,
): Promise<Result<string | undefined, CloudflareAiError | CloudflareVectorizeError | CacheError>> {
  const vector = embeddings.data[0];
  c.set("vector", vector);
  const startVectorize = performance.now();
  const query = await wrap(
    c.env.VECTORIZE_INDEX.query(vector, { topK: 1 }),
    (err) => new CloudflareVectorizeError({ message: err.message }),
  );
  c.set("vectorizeLatency", performance.now() - startVectorize);
  if (query.err) {
    return Err(query.err);
  }

  if (query.val.count === 0 || query.val.matches[0].score < MATCH_THRESHOLD) {
    c.set("cacheHit", false);
    return Ok(undefined);
  }

  const { cache } = c.get("services");

  const cacheStart = performance.now();
  const cacheKey = query.val.matches[0].id;
  const cached = await cache.completion.get(cacheKey);
  c.set("cacheLatency", performance.now() - cacheStart);
  if (cached.err) {
    return Err(cached.err);
  }
  c.set("cacheHit", !!cached.val);

  return Ok(cached.val?.content);
}

async function updateCache(
  c: Context,
  embeddings: AiTextEmbeddingsOutput,
  content: string,
): Promise<Result<void, CloudflareVectorizeError | CacheError>> {
  const { cache } = c.get("services");
  const id = await sha256(content);
  const cacheRes = await cache.completion.set(id, { id, content });
  if (cacheRes.err) {
    return Err(cacheRes.err);
  }
  const vector = embeddings.data[0];
  if (vector) {
    const vectorizeRes = await wrap(
      c.env.VECTORIZE_INDEX.insert([{ id, values: vector }]),
      (err) => new CloudflareVectorizeError({ message: err.message }),
    );
    if (vectorizeRes.err) {
      return Err(vectorizeRes.err);
    }
  }

  return Ok();
}
