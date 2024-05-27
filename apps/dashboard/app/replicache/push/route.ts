import { type ExtractTablesWithRelations, db, eq, schema } from "@/lib/db";
import { auth } from "@clerk/nextjs/server";
import { Transaction } from "@planetscale/database";
import type { MySqlTransaction } from "drizzle-orm/mysql-core";
import type {
  PlanetScalePreparedQueryHKT,
  PlanetscaleQueryResultHKT,
} from "drizzle-orm/planetscale-serverless";
import type { NextRequest } from "next/server";
import type { MutationV1, PushRequestV1 } from "replicache";

export const POST = async (req: NextRequest): Promise<Response> => {
  const { userId, orgId } = auth();
  if (!userId) {
    return new Response("Unauthorized", { status: 401 });
  }

  const ws = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, orgId ?? userId),
  });
  if (!ws) {
    return new Response("Workspace not found", { status: 404 });
  }
  const push = (await req.json()) as PushRequestV1;
  for (const mutation of push.mutations) {
    await db.transaction(async (tx) => {
      await processMutation(tx, ws.id, push.clientGroupID, mutation);
    });
  }

  return new Response();
};

async function processMutation(
  tx: MySqlTransaction<
    PlanetscaleQueryResultHKT,
    PlanetScalePreparedQueryHKT,
    typeof schema,
    ExtractTablesWithRelations<typeof schema>
  >,
  workspaceId: string,
  clientGroupId: string,
  mutation: MutationV1,
  error?: string,
) {
  let previous = await tx.query.replicacheServers.findFirst();
  if (!previous) {
    previous = { id: "id", version: 1 };
    await tx.insert(schema.replicacheServers).values(previous);
  }

  const nextVersion = previous.version + 1;
  const res = await tx.query.replicacheClients.findFirst({
    where: (table, { eq }) => eq(table.clientGroupId, clientGroupId),
    columns: {
      lastMutationId: true,
    },
  });
  const lastMutationId = res?.lastMutationId ?? 0;
  const nextMutationId = lastMutationId + 1;
  console.log("nextVersion", nextVersion, "nextMutationId", nextMutationId);
  // It's common due to connectivity issues for clients to send a
  // mutation which has already been processed. Skip these.
  if (mutation.id < nextMutationId) {
    console.log(mutation, nextMutationId);
    console.log(`Mutation ${mutation.id} has already been processed - skipping`);
    return;
  }

  // If the Replicache client is working correctly, this can never
  // happen. If it does there is nothing to do but return an error to
  // client and report a bug to Replicache.
  if (mutation.id > nextMutationId) {
    throw new Error(
      `Mutation ${mutation.id} is from the future - aborting. This can happen in development if the server restarts. In that case, clear appliation data in browser and refresh.`,
    );
  }
  if (error !== undefined) {
    throw new Error(error);
  }
  console.log("Processing mutation:", JSON.stringify(mutation));
  switch (mutation.name) {
    case "createApi": {
      const args = mutation.args as { id: string; name: string };
      await tx.insert(schema.apis).values({
        workspaceId,
        id: args.id,
        name: args.name,
      });
      break;
    }
    case "deleteApi": {
      const args = mutation.args as { id: string };
      await tx
        .update(schema.apis)
        .set({
          id: args.id,
          deletedAt: new Date(),
        })
        .where(eq(schema.apis.id, args.id));
      break;
    }
  }
  console.log("setting", mutation.clientID, "last_mutation_id to", nextMutationId);

  await tx
    .insert(schema.replicacheClients)
    .values({
      id: mutation.clientID,
      clientGroupId,
      lastMutationId: nextMutationId,
      version: nextVersion,
    })
    .onDuplicateKeyUpdate({
      set: {
        lastMutationId: nextMutationId,
        version: nextVersion,
      },
    });

  await tx
    .update(schema.replicacheServers)
    .set({
      id: "id",
      version: nextVersion,
    })
    .where(eq(schema.replicacheServers.id, "id"));
}
