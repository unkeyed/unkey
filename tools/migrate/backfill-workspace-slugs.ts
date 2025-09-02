import { eq, mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";

/**
 * Backfill script for workspace slugs
 *
 * This script will:
 * 1. Find all workspaces that don't have a slug
 * 2. Generate a slug from the workspace name following these rules:
 *    - Convert to lowercase
 *    - Replace spaces with hyphens
 *    - Remove all special characters except hyphens
 *    - Remove leading and trailing hyphens
 * 3. Update the database with the generated slug
 */

function generateSlug(name: string): string {
  return name
    .toLowerCase()
    .trim()
    .replace(/\s+/g, "-") // Replace spaces with hyphens
    .replace(/[^a-z0-9-]/g, "") // Remove all special characters except hyphens
    .replace(/-+/g, "-") // Replace multiple consecutive hyphens with single hyphen
    .replace(/^-+|-+$/g, ""); // Remove leading and trailing hyphens
}

async function main() {
  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  console.log("Starting workspace slug backfill migration...");

  let cursor = "";
  let processed = 0;
  let updated = 0;
  let errors = 0;

  do {
    // Find workspaces without slugs, ordered by ID for pagination
    const workspaces = await db.query.workspaces.findMany({
      where: (table, { isNull, gt, and }) => and(isNull(table.slug), gt(table.id, cursor)),
      limit: 1000,
      orderBy: (table, { asc }) => asc(table.id),
    });

    if (workspaces.length === 0) {
      break;
    }

    cursor = workspaces.at(-1)?.id ?? "";
    console.info({
      cursor,
      workspaces: workspaces.length,
      processed,
      updated,
      errors,
    });

    // Process each workspace
    for (const workspace of workspaces) {
      try {
        const slug = generateSlug(workspace.name);

        if (!slug) {
          console.warn(
            `Workspace ${workspace.id} (${workspace.name}) generated empty slug, skipping`,
          );
          continue;
        }

        // Check if slug already exists and find the next available number
        let finalSlug = slug;
        let counter = 1;

        while (true) {
          const existingWorkspace = await db.query.workspaces.findFirst({
            where: (table, { eq }) => eq(table.slug, finalSlug),
          });

          if (!existingWorkspace || existingWorkspace.id === workspace.id) {
            break; // Slug is available or belongs to this workspace
          }

          // Slug exists, try with next number
          finalSlug = `${slug}-${counter}`;
          counter++;
        }

        if (counter > 1) {
          console.warn(
            `Slug '${slug}' already exists, using '${finalSlug}' for workspace ${workspace.id}`,
          );
        }

        await db
          .update(schema.workspaces)
          .set({ slug: finalSlug })
          .where(eq(schema.workspaces.id, workspace.id));

        updated++;
        console.log(`Updated workspace ${workspace.id}: "${workspace.name}" -> "${slug}"`);
      } catch (error) {
        errors++;
        console.error(`Error processing workspace ${workspace.id}:`, error);
      }
    }

    processed += workspaces.length;
  } while (cursor);

  await conn.end();

  console.info("Migration completed!");
  console.info({
    totalProcessed: processed,
    totalUpdated: updated,
    totalErrors: errors,
  });
}

main().catch((error) => {
  console.error("Migration failed:", error);
  process.exit(1);
});
