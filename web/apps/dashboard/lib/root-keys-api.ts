import { z } from "zod";

const rootKeySchema = z.object({
  keyId: z.string(),
  name: z.string().nullable(),
  start: z.string(),
  end: z.string(),
  enabled: z.boolean(),
  createdAt: z.number(),
  lastUsedAt: z.number(),
  expires: z.number().nullable(),
  permissions: z.array(z.string()),
});

const rootKeySecretSchema = z.object({
  keyId: z.string(),
  key: z.string(),
});

const listResponseSchema = z.object({
  data: z.array(rootKeySchema),
  pagination: z.object({
    cursor: z.string().optional(),
    hasMore: z.boolean(),
  }),
});

const secretResponseSchema = z.object({ data: rootKeySecretSchema });

const apiErrorSchema = z.object({
  error: z.union([
    z.string(),
    z.object({
      detail: z.string().optional(),
      title: z.string().optional(),
    }),
  ]),
});

export type V2RootKey = z.infer<typeof rootKeySchema>;
export type RootKeySecret = z.infer<typeof rootKeySecretSchema>;

export type CreateRootKeyRequest = {
  name?: string;
  permissions: string[];
  expires?: number | null;
};

export type UpdateRootKeyRequest = {
  keyId: string;
  name?: string | null;
  enabled?: boolean;
  permissions?: string[];
};

export type DeleteRootKeyRequest = { keyId: string };

export type RerollRootKeyRequest = {
  keyId: string;
  expiration: number | null;
};

export const ROOT_KEYS_V2_QUERY_KEY = ["rootKeys", "v2"] as const;

async function post(path: string, body: unknown): Promise<unknown> {
  const response = await fetch(`/proxy/v2/rootKeys.${path}`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body),
  });

  const payload: unknown = await response.json();
  if (!response.ok) {
    const error = apiErrorSchema.safeParse(payload);
    const message = error.success
      ? typeof error.data.error === "string"
        ? error.data.error
        : (error.data.error.detail ?? error.data.error.title)
      : undefined;
    throw new Error(message ?? "Root key request failed");
  }

  return payload;
}

export async function listRootKeys(): Promise<V2RootKey[]> {
  const rootKeys: V2RootKey[] = [];
  const cursors = new Set<string>();
  let cursor: string | undefined;

  do {
    const body = cursor ? { limit: 100, cursor } : { limit: 100 };
    const page = listResponseSchema.parse(await post("listKeys", body));
    rootKeys.push(...page.data);

    if (page.pagination.hasMore && !page.pagination.cursor) {
      throw new Error("Root keys API returned another page without a cursor");
    }
    if (page.pagination.cursor && cursors.has(page.pagination.cursor)) {
      throw new Error("Root keys API returned a repeated cursor");
    }
    if (page.pagination.cursor) {
      cursors.add(page.pagination.cursor);
    }
    cursor = page.pagination.hasMore ? page.pagination.cursor : undefined;
  } while (cursor);

  return rootKeys;
}

export async function createRootKey(request: CreateRootKeyRequest): Promise<RootKeySecret> {
  return secretResponseSchema.parse(await post("createKey", request)).data;
}

export async function updateRootKey(request: UpdateRootKeyRequest): Promise<void> {
  await post("updateKey", request);
}

export async function deleteRootKey(request: DeleteRootKeyRequest): Promise<void> {
  await post("deleteKey", request);
}

export async function rerollRootKey(request: RerollRootKeyRequest): Promise<RootKeySecret> {
  return secretResponseSchema.parse(await post("rerollKey", request)).data;
}
