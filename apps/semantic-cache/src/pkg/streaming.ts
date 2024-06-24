import { OpenAIStream } from "ai";
import { streamSSE } from "hono/streaming";
import type { OpenAI } from "openai";

import type { Context } from "./hono/app";
import { OpenAIResponse, createCompletionChunk, extractWord, parseMessagesToString } from "./util";

import type { CacheError } from "@unkey/cache";
import { BaseError, Err, Ok, type Result, wrap } from "@unkey/error";
import { sha256 } from "@unkey/hash";

class OpenAiError extends BaseError {
  retry = false;
  name = OpenAiError.name;
}

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

    c.set("tokens", wordsWithWhitespace.length);
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
  const chatCompletion = await wrap(
    openai.chat.completions.create(requestOptions),
    (err) => new OpenAiError({ message: err.message }),
  );
  if (chatCompletion.err) {
    return c.text(chatCompletion.err.message, { status: 400 });
  }
  const responseStart = Date.now();
  const stream = OpenAIStream(chatCompletion.val);
  const [stream1, stream2, stream3] = triTee(stream);

  c.set("response", parseStream(stream3));

  c.executionCtx.waitUntil(
    (async () => {
      const s = await parseStream(stream2);
      await updateCache(c, embeddings.val, s, s.split(" ").length);
    })(),
  );

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
  const chatCompletion = await wrap(
    openai.chat.completions.create(request),
    (err) => new OpenAiError({ message: err.message }),
  );
  if (chatCompletion.err) {
    return c.text(chatCompletion.err.message, { status: 400 });
  }
  c.set("inferenceLatency", performance.now() - inferenceStart);
  const tokens = chatCompletion.val.usage?.completion_tokens ?? 0;
  c.set("tokens", tokens);

  const response = chatCompletion.val.choices.at(0)?.message.content || "";
  const { err: updateCacheError } = await updateCache(c, embeddings.val, response, tokens);
  if (updateCacheError) {
    logger.error("unable to update cache", {
      error: updateCacheError.message,
    });
  }

  c.set("response", Promise.resolve(response));
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
    c.env.VECTORIZE_INDEX.query(vector, { topK: 1, returnMetadata: true }),
    (err) => new CloudflareVectorizeError({ message: err.message }),
  );
  c.set("vectorizeLatency", performance.now() - startVectorize);
  if (query.err) {
    return Err(query.err);
  }

  const thresholdHeader = c.req.header("X-Min-Similarity");
  const treshold = thresholdHeader ? Number.parseFloat(thresholdHeader) : 0.9;

  if (query.val.count === 0 || query.val.matches[0].score < treshold) {
    c.set("cacheHit", false);
    c.res.headers.set("Unkey-Cache", "MISS");

    return Ok(undefined);
  }

  const response = query.val.matches[0].metadata?.response as string | undefined;
  c.set("tokens", query.val.matches[0].metadata?.tokens as number | undefined);

  c.set("cacheHit", true);
  c.res.headers.set("Unkey-Cache", "HIT");

  return Ok(response);
}

async function updateCache(
  c: Context,
  embeddings: AiTextEmbeddingsOutput,
  response: string,
  tokens: number,
): Promise<Result<void, CloudflareVectorizeError>> {
  const id = await sha256(response);
  const vector = embeddings.data[0];

  const vectorizeRes = await wrap(
    c.env.VECTORIZE_INDEX.insert([{ id, values: vector, metadata: { response, tokens } }]),
    (err) => new CloudflareVectorizeError({ message: err.message }),
  );
  if (vectorizeRes.err) {
    return Err(vectorizeRes.err);
  }

  return Ok();
}

function triTee<T>(
  stream: ReadableStream<T>,
): [ReadableStream<T>, ReadableStream<T>, ReadableStream<T>] {
  const [s1, tmp] = stream.tee();
  const [s2, s3] = tmp.tee();
  return [s1, s2, s3];
}
