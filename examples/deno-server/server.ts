import { verifyKey } from "https://unpkg.com/@unkey/api@latest/dist/index.mjs";

const port = Deno.env.get("PORT") || 8000;

console.log(`Launching Deno HTTP server on port: ${port}, url: http://0.0.0.0:${port} 🚀`);

const handler = async (req: Request): Promise<Response> => {
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
};

Deno.serve({ port }, handler);
