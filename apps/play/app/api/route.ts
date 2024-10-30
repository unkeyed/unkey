import { protectedApiRequestSchema } from "@/lib/schemas";

const bearerToken = process.env.PLAYGROUND_ROOT_KEY;

export async function POST(req: Request) {
  const body = await req.json();

  const payload = protectedApiRequestSchema.parse(body);

  const headers: RequestInit["headers"] = {
    Authorization: `Bearer ${bearerToken}`,
  };
  if (payload.method !== "GET") {
    headers["Content-Type"] = "application/json";
  }

  const res = await fetch(payload.url, {
    method: payload.method,
    headers,
    body: payload.jsonBody,
  });
  const data = await res.json();

  return Response.json(data);
}
