import { and, db, isNull } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../../trpc";

export const getAvailableKeyspaces = workspaceProcedure.query(async ({ ctx }) => {
  const keyspaces = await db.query.keyAuth
    .findMany({
      where: (table, { eq }) =>
        and(eq(table.workspaceId, ctx.workspace.id), isNull(table.deletedAtM)),
      columns: {
        id: true,
      },
      with: {
        api: {
          columns: {
            name: true,
            deletedAtM: true,
          },
        },
      },
    })
    .catch((err) => {
      console.error(err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Unable to load keyspaces.",
      });
    });

  return keyspaces.reduce(
    (acc, ks) => {
      if (ks.api && ks.api.deletedAtM === null) {
        acc[ks.id] = { id: ks.id, api: { name: ks.api.name } };
      }
      return acc;
    },
    {} as Record<string, { id: string; api: { name: string } }>,
  );
});
