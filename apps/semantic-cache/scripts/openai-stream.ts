const { config } = require("dotenv");
const OpenAI = require("openai");
config();
const openai = new OpenAI({
  apiKey: process.env.OPENAI_API_KEY,
  baseURL: "https://chronark.llm.unkey.io",
  //baseURL: "http://localhost:8787",
});

async function main() {
  console.info(process.argv[2]);
  const chatCompletion = await openai.chat.completions.create({
    messages: [
      {
        role: "user",
        content: process.argv[2],
      },
    ],
    model: "gpt-3.5-turbo",
    stream: true,
    user: "semantic",
    // noCache: true,
  });

  for await (const chunk of chatCompletion) {
    process.stdout.write(chunk.choices[0].delta.content);
  }
}

main();
