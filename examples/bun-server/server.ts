import { verifyKey } from "@unkey/api";

const port = process.env.PORT || 8000;

console.log(`Launching Bun HTTP server on port: ${port}, url: http://0.0.0.0:${port} 🚀`);

Bun.serve({
  async fetch(req) {
    const key = req.headers.get("Authorization")?.replace("Bearer ", "");
    if (!key) {
      return new Response("Unauthorized", { status: 401 });
    }

    const { result, error } = await verifyKey(key);
    if (error) {
      // This may happen on network errors
      // We already retry the request 5 times, but if it still fails, we return an error
      console.error(error);
      return Response.json("Internal Server Error", { status: 500 });
    }

    if (!result.valid) {
      return new Response("Unauthorized", { status: 401 });
    }

    return Response.json(result);
  },
  port: 8000,
});
