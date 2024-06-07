import { Analytics } from "@/pkg/analytics";
import { nanoid } from "nanoid";

function generateEvent() {
  return {
    requestId: nanoid(),
    time: Date.now(),
    latency: 100,
    gatewayId: "lgw_S7y7TdiEr2YbY8XRUoFJremy5UG",
    workspaceId: "ws_44ytdBUAzAwh6mdNmvgFZbcs5F8P",
    stream: true,
    tokens: 100,
    cache: true,
    model: "gpt-4",
    query: "write a poem about API keys",
    vector: [],
    response: "Hidden lines of code, \nSilent keys unlock the doors — \nSecrets of the web.",
  };
}

async function main() {
  try {
    const analytics = new Analytics({ tinybirdToken: process.env.TINYBIRD_TOKEN });
    const event = generateEvent();
    const result = await analytics.ingestLogs(event);
    console.info(result);
  } catch (error) {
    console.error(error);
  }
}

main();
