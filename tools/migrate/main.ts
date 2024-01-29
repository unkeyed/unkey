import { schema } from "@unkey/db";

import { connect } from "@planetscale/database";
import { newId } from "@unkey/id";
import { drizzle } from "drizzle-orm/planetscale-serverless";

async function main() {
  console.log("RUNNING");
  const db = drizzle(
    connect({
      host: process.env.DATABASE_HOST,
      username: process.env.DATABASE_USERNAME,
      password: process.env.DATABASE_PASSWORD,
    }),
    {
      schema,
    },
  );
  console.log("X");
  const oldRoles = await db.query.roles.findMany({
    with: { key: true },
  });
  let i = 0;
  for (const oldRole of oldRoles) {
    console.log("");
    console.log(++i, "/", oldRoles.length, oldRole.id);

    await db.transaction(async (tx) => {
      const existingPermission = await tx.query.permissions.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.workspaceId, oldRole.workspaceId), eq(table.name, oldRole.role)),
      });

      let permissionId: string = newId("permission");
      if (existingPermission) {
        permissionId = existingPermission.id;
      } else {
        await tx.insert(schema.permissions).values({
          id: permissionId,
          name: oldRole.role,
          workspaceId: oldRole.workspaceId,
        });
      }

      await tx
        .insert(schema.keysPermissions)
        .values({
          keyId: oldRole.key.id,
          permissionId,
          workspaceId: oldRole.workspaceId,
        })
        .onDuplicateKeyUpdate({
          set: {
            permissionId,
          },
        });
    });
  }
}

main();
