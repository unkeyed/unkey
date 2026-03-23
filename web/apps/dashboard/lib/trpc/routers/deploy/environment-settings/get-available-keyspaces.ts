import { and, db, isNull } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../../trpc";

export const getAvailableKeyspaces = workspaceProcedure.query(async ({ ctx }) => {
  try {
    const keyspaces = await db.query.keyAuth.findMany({
      where: (table, { eq }) =>
        and(eq(table.workspaceId, ctx.workspace.id), isNull(table.deletedAtM)),
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
  } catch (err) {
    if (err instanceof TRPCError) {
      throw err;
    }
    console.error(err);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Unable to load keyspaces.",
    });
  }
});
