import { RateLimitExceededException, WorkOS } from "@workos-inc/node";
import pLimit from "p-limit";
import { createClerkClient, Organization } from "@clerk/clerk-sdk-node";
import { eq, drizzle, schema, type Database } from "@unkey/db";
import { Client } from "@planetscale/database";

const workos = new WorkOS(process.env.WORKOS_API_KEY!);

export const db: Database = drizzle(
  new Client({
    host: process.env.DATABASE_HOST,
    username: process.env.DATABASE_USERNAME,
    password: process.env.DATABASE_PASSWORD,

    fetch: (url: string, init: any) => {
      (init as any).cache = undefined; // Remove cache header
      const u = new URL(url);
      // set protocol to http if localhost for CI testing
      if (u.host.includes("localhost")) {
        u.protocol = "http";
      }
      return fetch(u, init);
    },
  }),
  {
    schema,
  }
);

const clerk = createClerkClient({
  secretKey: process.env.CLERK_SECRET_KEY!,
});

const limit = pLimit(10); // 10 operations per second
const PAGE_SIZE = 500;

async function getClerkOrganizations() {
  let hasMore = true;
  let offset = 0;
  const allOrgs = [];
  while (hasMore) {
    const response = await limit(() =>
      clerk.organizations.getOrganizationList({
        limit: PAGE_SIZE,
        offset: offset,
      })
    );

    allOrgs.push(...response.data);

    if (response.data.length < PAGE_SIZE) {
      hasMore = false;
    } else {
      offset += PAGE_SIZE;
    }
  }
  return allOrgs;
}

async function getClerkMemberships(organizationId: string) {
  let hasMore = true;
  let offset = 0;
  const orgMemberShip = [];
  while (hasMore) {
    const response = await limit(() =>
      clerk.organizations.getOrganizationMembershipList({
        organizationId,
        limit: PAGE_SIZE,
        offset: offset,
      })
    );
    orgMemberShip.push(...response.data);
    if (response.data.length < PAGE_SIZE) {
      hasMore = false;
    } else {
      offset += PAGE_SIZE;
    }
  }
  return orgMemberShip;
}

async function findWorkOSUserByClerkId(userId: string) {
  const user = await workos.userManagement.getUserByExternalId(userId);
  return user;
}

const migrateOrg = async (org: Organization) => {
  try {
    const orgExists = await workos.organizations
      .getOrganizationByExternalId(org.id)
      .catch(() => {});
    if (orgExists) {
      console.log(`Organization ${org.name} already exists, skipping`);
      return orgExists;
    }

    const workspaceCheck = await db
      .select()
      .from(schema.workspaces)
      .where(eq(schema.workspaces.clerkTenantId, org.id));

    if (workspaceCheck.length === 0) {
      console.log(`No workspaces found for organization ${org.name}, skipping`);
      return null;
    }
    const workosOrg = await workos.organizations.createOrganization({
      name: org.name,
      externalId: org.id, // Store Clerk org ID as external_id
    });

    console.log(`Created WorkOS organization: ${org.name}`);
    if (!workosOrg) {
      return null;
    }

    const updateDB = await db
      .update(schema.workspaces)
      .set({
        orgId: workosOrg.id,
      })
      .where(eq(schema.workspaces.clerkTenantId, org.id));

    if (!updateDB.rowsAffected || updateDB.rowsAffected === 0) {
      throw new Error(`Failed to update database for Clerk Org ${org.id}`);
    }

    // Get all memberships for this organization
    const memberships = await getClerkMemberships(org.id);

    // Add members to the organization
    const results = await Promise.all(
      memberships.map(async (membership) => {
        try {
          const externalId = membership.publicUserData?.userId;
          if (!externalId) {
            return null;
          }
          const workosUser = await findWorkOSUserByClerkId(externalId);

          if (workosUser) {
            const result =
              await workos.userManagement.createOrganizationMembership({
                organizationId: workosOrg.id,
                userId: workosUser.id,
                roleSlug: membership.role.toLowerCase(),
              });

            if (!result) {
              console.error(
                `Failed to add user ${workosUser.email} to organization ${org.name}`
              );
              return null;
            }
            console.log(
              `Added user ${workosUser.email} to organization ${org.name}`
            );
            return result;
          } else {
            console.log(`User not found in WorkOS: ${externalId}`);
            return null;
          }
        } catch (error) {
          console.error(`Error adding member to organization: ${error}`);
          return null;
        }
      })
    );
    return results;
  } catch (error) {
    if (error instanceof RateLimitExceededException) {
      console.warn(`Rate limit hit, retrying org: ${org.id}`);
      // Re-throw to trigger retry
      throw error;
    }

    if (error instanceof Error) {
      console.error(`Error importing org ${org.id}: ${error.message}`);
    } else {
      console.error(`Error importing org ${org.id}: Unknown error type`, error);
    }
  }
};

const migrateOrganizations = async () => {
  const organizations = await getClerkOrganizations();
  const limit = pLimit(5); // Limit concurrent operations
  const results = [];
  for (let i = 0; i < organizations.length; i += 10) {
    const batch = organizations.slice(i, i + 10);
    const batchPromises = batch.map((org) => limit(() => migrateOrg(org)));

    try {
      const batchResults = await Promise.all(batchPromises);
      results.push(...batchResults.filter((r) => r !== null));
      console.log(
        `Completed batch ${i / 10 + 1} of ${Math.ceil(
          organizations.length / 10
        )}`
      );

      if (i + 10 < organizations.length) {
        await new Promise((resolve) => setTimeout(resolve, 1000));
      }
    } catch (error) {
      console.error(`Error processing batch ${i / 10 + 1}:`, error);
    }
  }

  console.log(
    `Import completed. Successfully imported ${results.length} out of ${organizations.length} organizations`
  );
  return results;
};

migrateOrganizations().then(() => process.exit(0));