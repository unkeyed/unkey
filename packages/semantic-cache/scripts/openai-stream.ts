import "dotenv/config";
import OpenAI from "openai";

const openai = new OpenAI({
  apiKey: process.env.OPENAI_API_KEY,
  baseURL: "http://localhost:59061",
  // baseURL: "https://llm.unkey.dev",
});

async function main() {
  const chatCompletion = await openai.chat.completions.create({
    messages: [
      {
        role: "user",
        content: process.argv[2],
      },
    ],
    model: "gpt-4",
    stream: true,
    // noCache: true,
  });

  for await (const chunk of chatCompletion) {
    console.log(chunk.choices[0].delta.content);
  }
}

main();
