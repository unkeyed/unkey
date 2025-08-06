import { getAuth } from "@/lib/auth";
import { and, db, eq, isNull } from "@/lib/db";
import { getAllKeys } from "@/lib/trpc/routers/api/keys/query-api-keys/get-all-keys";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { apis } from "@unkey/db/src/schema";
import { notFound } from "next/navigation";

export type ApiLayoutData = {
  currentApi: {
    id: string;
    name: string;
    workspaceId: string;
    keyAuthId: string | null;
    keyspaceDefaults: {
      prefix?: string;
      bytes?: number;
    } | null;
  };
  workspaceApis: {
    id: string;
    name: string;
  }[];
};

export const fetchApiAndWorkspaceDataFromDb = async (apiId: string): Promise<ApiLayoutData> => {
  const { orgId } = await getAuth();
  if (!apiId || !orgId) {
    console.error("fetchApiLayoutDataFromDb: apiId or orgId is missing");
    notFound();
  }

  const currentApi = await db.query.apis.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.id, apiId), isNull(table.deletedAtM)),
    with: {
      workspace: {
        columns: {
          id: true,
          orgId: true,
        },
      },
      keyAuth: {
        columns: {
          defaultPrefix: true,
          defaultBytes: true,
        },
      },
    },
    columns: {
      id: true,
      name: true,
      workspaceId: true,
      keyAuthId: true,
    },
  });

  if (!currentApi || currentApi.workspace.orgId !== orgId) {
    console.warn(`DB Validation failed: API ${apiId} not found or org mismatch for org ${orgId}`);
    notFound();
  }

  const workspaceId = currentApi.workspaceId;

  const workspaceApis = await db
    .select({ id: apis.id, name: apis.name })
    .from(apis)
    .where(and(eq(apis.workspaceId, workspaceId), isNull(apis.deletedAtM)))
    .orderBy(apis.name);

  return {
    currentApi: {
      id: currentApi.id,
      name: currentApi.name,
      workspaceId: currentApi.workspaceId,
      keyAuthId: currentApi.keyAuthId,
      keyspaceDefaults: {
        prefix: currentApi.keyAuth?.defaultPrefix || undefined,
        bytes: currentApi.keyAuth?.defaultBytes || undefined,
      },
    },
    workspaceApis,
  };
};

export async function getKeyDetails(
  keyId: string,
  keyspaceId: string,
  workspaceId: string,
): Promise<KeyDetails | null> {
  const result = await getAllKeys({
    keyspaceId,
    workspaceId,
    filters: {
      keyIds: [{ operator: "is", value: keyId }],
    },
    limit: 1,
  });

  // If no keys found, return null
  if (result.keys.length === 0) {
    return null;
  }

  // Return the first (and only) key
  return result.keys[0];
}
