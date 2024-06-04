import "dotenv/config";
import OpenAI from "openai";

const openai = new OpenAI({
  apiKey: process.env.OPENAI_API_KEY,
  baseURL: "http://localhost:8787",
  // baseURL: "https://llm.unkey.dev",
});

async function main() {
  const chatCompletion = await openai.chat.completions.create({
    messages: [
      {
        role: "user",
        content: process.argv[2], // tsx openai-stream.ts 'hello'
      },
    ],
    model: "gpt-4",
    stream: true,
    user: "semantic", // pass a user for testing auth
    // noCache: true, // disable caching
  });

  for await (const chunk of chatCompletion) {
    console.info(chunk.choices[0].delta.content);
  }
}

main();
