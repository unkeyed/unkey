import { nanoid } from "nanoid";
export function createCompletionChunk(content, stop = false) {
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
export function OpenAIResponse(content) {
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
export function extractWord(chunk) {
  const match = chunk.match(/"([^"]*)"/);
  return match ? match[1] : "";
}
export function parseMessagesToString(messages) {
  return messages.at(-1)?.content || "";
}
export async function getEmbeddings(c, messages) {
  const embeddingsRequest = await c.env.AI.run("@cf/baai/bge-small-en-v1.5", {
    text: messages,
  });
  return embeddingsRequest.data[0];
}
