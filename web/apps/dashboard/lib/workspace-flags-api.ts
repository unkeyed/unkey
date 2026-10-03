import { z } from "zod";

const flagSchema = z.object({
  slug: z.string(),
  description: z.string(),
  value: z.boolean(),
  defaultValue: z.boolean(),
  hasOverride: z.boolean(),
  allowOptIn: z.boolean(),
  allowOptOut: z.boolean(),
});

export type WorkspaceFlag = z.infer<typeof flagSchema>;

export async function listWorkspaceFlags(signal?: AbortSignal): Promise<WorkspaceFlag[]> {
  return z.object({ data: z.array(flagSchema) }).parse(await post("listFlags", {}, signal)).data;
}

export async function setWorkspaceFlagOverride(
  slug: string,
  value: boolean,
): Promise<WorkspaceFlag> {
  return z.object({ data: flagSchema }).parse(await post("setOverride", { slug, value })).data;
}

export async function removeWorkspaceFlagOverride(slug: string): Promise<WorkspaceFlag> {
  return z.object({ data: flagSchema }).parse(await post("removeOverride", { slug })).data;
}

async function post(path: string, body: unknown, signal?: AbortSignal): Promise<unknown> {
  const response = await fetch(`/proxy/v2/flags.${path}`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body),
    signal,
  });
  if (!response.ok) {
    const error = z
      .object({ error: z.union([z.string(), z.object({ detail: z.string() })]) })
      .safeParse(await response.json());
    throw new Error(
      error.success
        ? typeof error.data.error === "string"
          ? error.data.error
          : error.data.error.detail
        : "We couldn't complete the platform feature request. Try again.",
    );
  }
  return response.json();
}
