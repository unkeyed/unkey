import { and, eq, isNull, mysqlDrizzle, schema } from "@unkey/db";
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
  // 1) normalize Unicode, strip diacritics; 2) canonicalize; 3) trim edges
  const base = name
    .toLowerCase()
    .normalize("NFKD")
    .trim()
    .replace(/\s+/g, "-") // Replace spaces with hyphens
    .replace(/[^a-z0-9-]/g, "") // Remove all special characters except hyphens
    .replace(/-+/g, "-") // Collapse multiple hyphens
    .replace(/^-+|-+$/g, ""); // Trim leading/trailing hyphens

  // reserve space for a "-<n>" suffix later; limit to 61 chars for the base
  const MAX_BASE = 61;
  const trimmed = base.slice(0, MAX_BASE);

  return trimmed;
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

          // Parse existing slug to extract base and numeric suffix
          const slugMatch = finalSlug.match(/^(.+?)(?:-(\d+))?$/);
          const base = slugMatch?.[1] || slug;
          const existingSuffix = slugMatch?.[2] ? Number.parseInt(slugMatch[2], 10) : 0;

          // Increment the numeric suffix
          const nextSuffix = existingSuffix + 1;

          // Calculate available space for base portion
          // Reserve space for "-<number>" where number can be up to 999999
          const suffixLength = `-${nextSuffix}`.length;
          const maxBaseLength = 64 - suffixLength;

          // Truncate base if needed to fit within total length limit
          const truncatedBase = base.slice(0, maxBaseLength);

          // Ensure we don't end with a hyphen
          const cleanBase = truncatedBase.replace(/-+$/, "");

          finalSlug = `${cleanBase}-${nextSuffix}`;
          counter++;
        }

        if (counter > 1) {
          console.warn(
            `Slug '${slug}' already exists, using '${finalSlug}' for workspace ${workspace.id}`,
          );
        }

        // Update only rows where slug IS NULL to prevent TOCTOU race conditions
        await db
          .update(schema.workspaces)
          .set({ slug: finalSlug })
          .where(and(eq(schema.workspaces.id, workspace.id), isNull(schema.workspaces.slug)));

        // Check if the update actually affected a row by verifying the database state
        const updatedWorkspace = await db.query.workspaces.findFirst({
          where: (table, { eq }) => eq(table.id, workspace.id),
        });
        const rowUpdated = updatedWorkspace?.slug === finalSlug;

        if (rowUpdated) {
          updated++;
          console.log(`Updated workspace ${workspace.id}: "${workspace.name}" -> "${finalSlug}"`);
        } else {
          console.warn(
            `Workspace ${workspace.id} was not updated (slug may have been set concurrently)`,
          );
        }
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
