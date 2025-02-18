"use server";

import { db } from "@/lib/db";
import { Filter } from "./filter";

export const RootKeyFilter: React.FC<{ workspaceId: string }> = async ({ workspaceId }) => {
  const rootKeys = await db.query.keys.findMany({
    where: (table, { eq }) => eq(table.forWorkspaceId, workspaceId),

    columns: {
      id: true,
      name: true,
    },
  });

  return (
    <Filter
      param="rootKeys"
      title="Root Keys"
      options={rootKeys.map((k) => ({
        label: k.id ?? k.id,
        value: k.id,
      }))}
    />
  );
};
