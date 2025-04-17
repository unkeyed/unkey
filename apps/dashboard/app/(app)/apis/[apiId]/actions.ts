import { getOrgId } from "@/lib/auth";
import { and, db, eq, isNull } from "@/lib/db";
import { apis } from "@unkey/db/src/schema";
import { notFound } from "next/navigation";

export type ApiLayoutData = {
  currentApi: {
    id: string;
    name: string;
    workspaceId: string;
    keyAuthId: string | null;
  };
  workspaceApis: {
    id: string;
    name: string;
  }[];
};

export const fetchApiAndWorkspaceDataFromDb = async (apiId: string): Promise<ApiLayoutData> => {
  const orgId = await getOrgId();
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
    },
    workspaceApis,
  };
};
