import { Analytics } from "../src/analytics";

async function main() {
  const analytics = new Analytics({ tinybirdToken: process.env.TINYBIRD_TOKEN as string });
  console.info("Analytics", analytics);

  const event: AnalyticsEvent = {
    timestamp: new Date().toISOString(),
    model: "gpt-4",
    stream: true,
    query: "hello",
    vector: [0],
    response: "hello there!",
    cache: true,
    timing: 100,
    tokens: 100,
    requestId: "123",
  };

  await analytics.ingestLogs(event);
}

main();
