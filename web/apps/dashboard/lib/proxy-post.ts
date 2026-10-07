import { z } from "zod";

const apiErrorSchema = z.object({
  error: z.union([
    z.string(),
    z.object({
      detail: z.string().optional(),
      title: z.string().optional(),
    }),
  ]),
});

export async function proxyPost(
  route: string,
  body: unknown,
  fallbackMessage: string,
  signal?: AbortSignal,
): Promise<unknown> {
  const response = await fetch(`/proxy/v2/${route}`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body),
    signal,
  });

  const payload: unknown = await response.json();
  if (!response.ok) {
    const error = apiErrorSchema.safeParse(payload);
    const message = error.success
      ? typeof error.data.error === "string"
        ? error.data.error
        : (error.data.error.detail ?? error.data.error.title)
      : undefined;
    throw new Error(message ?? fallbackMessage);
  }

  return payload;
}
