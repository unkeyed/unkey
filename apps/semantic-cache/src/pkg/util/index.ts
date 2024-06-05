import type { Context } from "hono";
import { nanoid } from "nanoid";
import type { ChatCompletionMessageParam } from "openai/resources/chat/completions";

export function createCompletionChunk(content: string, stop = false) {
  return {
    id: `chatcmpl-${nanoid()}`,
    object: "chat.completion.chunk",
    created: new Date().toISOString(),
    model: "gpt-4",
    choices: [
      {
        delta: {
          content,
        },
        index: 0,
        logprobs: null,
        finish_reason: stop ? "stop" : null,
      },
    ],
  };
}

export function OpenAIResponse(content: string) {
  return {
    choices: [
      {
        message: {
          content,
        },
      },
    ],
  };
}

export function extractWord(chunk: string): string {
  const match = chunk.match(/"([^"]*)"/);
  return match ? match[1] : "";
}

export function parseMessagesToString(messages: Array<ChatCompletionMessageParam>) {
  return (messages.at(-1)?.content || "") as string;
}

export async function getEmbeddings(c: Context, messages: string) {
  const embeddingsRequest = await c.env.AI.run("@cf/baai/bge-small-en-v1.5", {
    text: messages,
  });

  return embeddingsRequest.data[0];
}
