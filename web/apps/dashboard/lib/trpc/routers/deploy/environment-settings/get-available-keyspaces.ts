import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../../trpc";

export const getAvailableKeyspaces = workspaceProcedure.query(async ({ ctx }) => {
  const keyspaces = await db.query.keyAuth
    .findMany({
      where: { workspaceId: ctx.workspace.id, deletedAtM: { isNull: true } },
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
      acc[ks.id] = ks;
      return acc;
    },
    {} as Record<string, { id: string; api: { name: string } }>,
  );
});
