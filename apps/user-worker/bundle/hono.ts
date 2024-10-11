import { Hono } from "hono";

const app = new Hono();

app.get("/hello", () => {
  return new Response("world something else I guesss");
});

export default app;
