import { Hono } from "hono";

const app = new Hono();

app.get("/hello", () => {
  return new Response("world");
});

export default app;
