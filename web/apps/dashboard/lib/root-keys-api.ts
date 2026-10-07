import { proxyPost } from "@/lib/proxy-post";
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

export type RerollRootKeyRequest = {
  keyId: string;
  expiration: number | null;
};

export const rootKeysV2QueryKeys = {
  all: ["rootKeys", "v2"] as const,
  workspace: (workspaceId: string) => [...rootKeysV2QueryKeys.all, workspaceId] as const,
};

function post(path: string, body: unknown, signal?: AbortSignal): Promise<unknown> {
  return proxyPost(`rootKeys.${path}`, body, "Root key request failed", signal);
}

export async function listRootKeys(signal?: AbortSignal): Promise<V2RootKey[]> {
  const rootKeys: V2RootKey[] = [];
  const cursors = new Set<string>();
  let cursor: string | undefined;

  do {
    const body = cursor ? { limit: 100, cursor } : { limit: 100 };
    const page = listResponseSchema.parse(await post("listKeys", body, signal));
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

export async function deleteRootKey(request: { keyId: string }): Promise<void> {
  await post("deleteKey", request);
}

export async function rerollRootKey(request: RerollRootKeyRequest): Promise<RootKeySecret> {
  return secretResponseSchema.parse(await post("rerollKey", request)).data;
}
