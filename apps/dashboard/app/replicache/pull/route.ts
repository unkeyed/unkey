import { db } from "@/lib/db";
import { auth } from "@clerk/nextjs/server";
import type { PatchOperation, PullRequestV1, PullResponseV1 } from "replicache";

export const POST = async (req: Request): Promise<Response> => {
  const { userId, orgId } = await auth();
  if (!userId) {
    return new Response("Unauthorized", { status: 401 });
  }

  const ws = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, orgId ?? userId),
  });
  if (!ws) {
    return new Response("Workspace not found", { status: 404 });
  }
  const pull = (await req.json()) as PullRequestV1;
  const fromVersion = Number(pull.cookie ?? 0);

  const responseBody = await db.transaction(async (tx) => {
    const current = await tx.query.replicacheServers.findFirst();
    if (fromVersion > current!.version) {
      throw new Error(
        `fromVersion ${fromVersion} is from the future - aborting. This can happen in development if the server restarts. In that case, clear appliation data in browser and refresh.`,
      );
    }
    const lastMutationIdChanges = await tx.query.replicacheClients.findMany({
      where: (table, { and, eq, gt }) =>
        and(eq(table.clientGroupId, pull.clientGroupID), gt(table.version, fromVersion)),
      columns: {
        id: true,
        lastMutationId: true,
      },
    });

    const changes = await tx.query.apis.findMany({
      where: (table, { and, gte, eq }) => and(eq(table.workspaceId, ws.id), gte(table.version, 0)),
    });
    console.log(JSON.stringify({ changes }, null, 2));

    const patch: PatchOperation[] = [];
    for (const api of changes) {
      if (api.deletedAt) {
        patch.push({
          op: "del",
          key: `api/${api.id}`,
        });
      } else {
        patch.push({
          op: "put",
          key: `api/${api.id}`,
          value: {
            id: api.id,
            name: api.name,
          },
        });
      }
    }
    const body: PullResponseV1 = {
      lastMutationIDChanges: lastMutationIdChanges.reduce(
        (acc, c) => {
          acc[c.id] = c.lastMutationId;
          return acc;
        },
        {} as Record<string, number>,
      ),
      cookie: pull.cookie,
      patch,
    };
    return body;
  });
  console.log("responseBody", responseBody);
  return Response.json(responseBody);
};
