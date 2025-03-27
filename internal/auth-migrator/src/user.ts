import {
  WorkOS,
  RateLimitExceededException,
  User as WorkOSUser,
} from "@workos-inc/node";
import pLimit from "p-limit";
import { createClerkClient, User } from "@clerk/clerk-sdk-node";
import { eq, drizzle, schema, type Database } from "@unkey/db";
import { Client } from "@planetscale/database";

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

const workos = new WorkOS(process.env.WORKOS_API_KEY!);

const clerk = createClerkClient({
  secretKey: process.env.CLERK_SECRET_KEY!,
});

const limit = pLimit(10); // 10 operations per second

const getUsers = async () => {
  console.log("Fetching users from Clerk...");
  const allUsers = [];
  let hasMore = true;
  let offset = 0;
  const PAGE_SIZE = 500;

  while (hasMore) {
    const response = await limit(() =>
      clerk.users.getUserList({
        limit: PAGE_SIZE,
        offset: offset,
      })
    );

    allUsers.push(...response.data);
    console.log(`Fetched ${response.data.length} users`);
    if (response.data.length < PAGE_SIZE) {
      hasMore = false;
    } else {
      offset += PAGE_SIZE;
    }
  }
  return allUsers;
};

const createOrganizationForUser = async (user: WorkOSUser) => {
  const org = await workos.organizations.createOrganization({
    name: "Personal Workspace ",
    externalId: user.externalId,
  });
  if (!org) {
    throw new Error(`Failed to create organization for user ${user.id}`);
  }
  const orgId = org.id;

  const createOrgMember =
    await workos.userManagement.createOrganizationMembership({
      organizationId: orgId,
      userId: user.id,
      roleSlug: "admin",
    });
  if (!createOrgMember) {
    throw new Error(`Failed to create organization member for user ${user.id}`);
  }
  return org;
};

const importUser = async (user: User) => {
  try {
    if (!user.primaryEmailAddress) {
      console.error(`User ${user.id} has no primary email address, skipping`);
      return null;
    }

    try {
      const matchingUsers = await workos.userManagement.listUsers({
        email: user.primaryEmailAddress!.emailAddress.toLowerCase(),
      });

      if (matchingUsers.data.length === 1) {
        console.log(
          `User ${user.primaryEmailAddress!.emailAddress.toLowerCase()} already exists, skipping`
        );
        return matchingUsers.data[0];
      }
    } catch (lookupError) {
      console.error(
        `Error looking up existing user ${user.primaryEmailAddress!.emailAddress.toLowerCase()}:`,
        lookupError
      );
    }

    const result = await workos.userManagement.createUser({
      emailVerified: true,
      email: user.primaryEmailAddress.emailAddress,
      firstName: user.firstName ? user.firstName : undefined,
      lastName: user.lastName ? user.lastName : undefined,
      externalId: user.id,
    });

    if (!result) {
      throw new Error(`Failed to create user ${user.id}`);
    }
    const workspaceCheck = await db
      .select()
      .from(schema.workspaces)
      .where(eq(schema.workspaces.clerkTenantId, user.id));
    if (workspaceCheck.length === 0) {
      console.log(
        `No workspace found for user, ${user.id} so skipping org creation`
      );
    } else {
      const org = await createOrganizationForUser(result);
      if (!org) {
        throw new Error(`Failed to create organization for user ${user.id}`);
      }
      console.log(`Successfully created organization for user ${user.id}`);
      console.log(`Upserting organization Id into DB ${org.id}`);
      const updateDB = await db
        .update(schema.workspaces)
        .set({
          orgId: org.id,
        })
        .where(eq(schema.workspaces.clerkTenantId, user.id));

      if (!updateDB.rowsAffected || updateDB.rowsAffected === 0) {
        console.error(
          `Failed to update database for user ${user.id}, here is the organization id ${org.id}`
        );
        throw new Error(
          `Failed to update database for user ${user.id}, here is the organization id ${org.id}`
        );
      }

      console.log(
        `Successfully imported user: ${user.primaryEmailAddress.emailAddress.toLowerCase()}`
      );
    }
    return result;
  } catch (error) {
    if (error instanceof RateLimitExceededException) {
      console.warn(
        `Rate limit hit, retrying user: ${user.primaryEmailAddress!.emailAddress.toLowerCase()}`
      );
      // Re-throw to trigger retry
      throw error;
    }

    if (error instanceof Error) {
      console.error(
        `Error importing user ${user.primaryEmailAddress!.emailAddress.toLowerCase()}: ${
          error.message
        }`
      );
    } else {
      console.error(
        `Error importing user ${user.primaryEmailAddress!.emailAddress.toLowerCase()}: Unknown error type`,
        error
      );
    }

    return null;
  }
};

const importUsers = async () => {
  const users = await getUsers();
  console.log(`Found ${users.length} users to import`);

  const results = [];
  for (let i = 0; i < users.length; i += 10) {
    const batch = users.slice(i, i + 10);
    const batchPromises = batch.map((user) => limit(() => importUser(user)));

    try {
      const batchResults = await Promise.all(batchPromises);
      results.push(...batchResults.filter((r) => r !== null));
      console.log(
        `Completed batch ${i / 10 + 1} of ${Math.ceil(users.length / 10)}`
      );

      // Add a small delay between batches to prevent overwhelming the API
      if (i + 10 < users.length) {
        await new Promise((resolve) => setTimeout(resolve, 1000));
      }
    } catch (error) {
      console.error(`Error processing batch ${i / 10 + 1}:`, error);
    }
  }

  console.log(
    `Import completed. Successfully imported ${results.length} out of ${users.length} users`
  );
  return results;
};

importUsers().then(() => process.exit(0));
