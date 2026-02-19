import { db } from "@/lib/db";
import { workspaceProcedure } from "../../../trpc";

export const getAvailableKeyspaces = workspaceProcedure.query(async ({ ctx }) => {
  const keyspaces = await db.query.keyAuth.findMany({
    where: (table, { eq }) => eq(table.workspaceId, ctx.workspace.id),
    columns: {
      id: true,
    },
    with: {
      api: {
        columns: {
          name: true,
        },
      },
    },
  });

  return keyspaces.reduce(
    (acc, ks) => {
      acc[ks.id] = ks;
      return acc;
    },
    {} as Record<string, { id: string; api: { name: string } }>,
  );
});
