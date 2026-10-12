import { type SQL, and, db, eq, inArray } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { identities, keys } from "@unkey/db/src/schema";

const MAX_IDENTITY_FILTER_IDS = 100;

export async function resolveIdentityFilter({
  workspaceId,
  condition,
}: {
  workspaceId: string;
  condition: SQL;
}) {
  const matchingIdentities = () =>
    db
      .select({ id: identities.id })
      .from(identities)
      .where(and(eq(identities.workspaceId, workspaceId), condition));

  const matches = await matchingIdentities()
    .limit(MAX_IDENTITY_FILTER_IDS + 1)
    .catch((error) => {
      console.error("Database query error:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retrieve API information.",
      });
    });

  if (matches.length > MAX_IDENTITY_FILTER_IDS) {
    return inArray(keys.identityId, matchingIdentities());
  }

  return inArray(
    keys.identityId,
    matches.map((identity) => identity.id),
  );
}
