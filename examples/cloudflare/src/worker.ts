import { verifyKey } from "@unkey/api";

export default {
  async fetch(request: Request): Promise<Response> {
    // Grab the key from the authorization header
    const authHeader = request.headers.get("Authorization");
    if (!authHeader) {
      return new Response("Unauthorized (No key)", { status: 401 });
    }
    const key = authHeader.replace("Bearer ", "");

    const { result, error } = await verifyKey(key);
    if (error) {
      console.error(error.message);
      return new Response("Internal Server Error", { status: 500 });
    }
    if (!result.valid) {
      return new Response("Unauthorized", { status: 401 });
    }

    // proceed to handle the request
    // since this is a demo, we just return the result
    return Response.json(result);
  },
};
