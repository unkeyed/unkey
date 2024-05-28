import { type ExtractTablesWithRelations, db, eq, schema } from "@/lib/db";
import { auth } from "@clerk/nextjs/server";
import type { MySqlTransaction } from "drizzle-orm/mysql-core";
import type {
  PlanetScalePreparedQueryHKT,
  PlanetscaleQueryResultHKT,
} from "drizzle-orm/planetscale-serverless";
import type { NextRequest } from "next/server";
import type { MutationV1, PushRequestV1 } from "replicache";

type Transaction = MySqlTransaction<
  PlanetscaleQueryResultHKT,
  PlanetScalePreparedQueryHKT,
  typeof schema,
  ExtractTablesWithRelations<typeof schema>
>;

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
    try {
      await db.transaction(async (tx) => {
        await processMutation(tx, ws.id, push.clientGroupID, mutation);
      });
    } catch (e) {
      await db.transaction(async (tx) => {
        await processMutation(tx, ws.id, push.clientGroupID, mutation, (e as Error).message);
      });
    }
  }

  return new Response();
};

async function processMutation(
  tx: Transaction,
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

  const lastMutationId = await getlastMutationId(tx, mutation.clientID);
  const nextMutationId = lastMutationId + 1;
  // It's common due to connectivity issues for clients to send a
  // mutation which has already been processed. Skip these.
  if (mutation.id < nextMutationId) {
    console.debug(`Mutation ${mutation.id} has already been processed - skipping`);
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
  if (error === undefined) {
    switch (mutation.name) {
      case "createApi": {
        const args = mutation.args as { id: string; name: string };
        await tx.insert(schema.apis).values({
          workspaceId,
          id: args.id,
          name: args.name,
          version: nextVersion,
        });
        break;
      }
      case "deleteApi": {
        const args = mutation.args as { id: string };
        await tx
          .update(schema.apis)
          .set({
            id: args.id,
            version: nextVersion,
            deletedAt: new Date(),
          })
          .where(eq(schema.apis.id, args.id));
        break;
      }
    }
  } else {
    console.error("Handling error from mutation", JSON.stringify(mutation), error);
  }
  await setLastMutationId(tx, mutation.clientID, clientGroupId, mutation.id, nextVersion);

  await tx
    .update(schema.replicacheServers)
    .set({
      version: nextVersion,
    })
    .where(eq(schema.replicacheServers.id, "id"));
}

async function getlastMutationId(tx: Transaction, clientId: string): Promise<number> {
  const client = await tx.query.replicacheClients.findFirst({
    where: (table, { eq }) => eq(table.id, clientId),
    columns: {
      lastMutationId: true,
    },
  });
  return client?.lastMutationId ?? 0;
}

async function setLastMutationId(
  tx: Transaction,
  clientId: string,
  clientGroupId: string,
  mutationId: number,
  version: number,
): Promise<void> {
  await tx
    .insert(schema.replicacheClients)
    .values({
      id: clientId,
      clientGroupId,
      lastMutationId: mutationId,
      version,
    })
    .onDuplicateKeyUpdate({
      set: {
        clientGroupId,
        lastMutationId: mutationId,
        version,
      },
    });
}
