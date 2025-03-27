import {
  WorkOS,
  RateLimitExceededException,
  WorkOSResponseError,
  User as WorkOSUser,
} from "@workos-inc/node";
import pLimit from "p-limit";

const workos = new WorkOS(process.env.WORKOS_API_KEY);
const limit = pLimit(10); // Process 10 users concurrently

type OrgData = {
  id: string;
  name: string;
};

async function getWorkOSOrganizations() {
  let list = await workos.organizations.listOrganizations({
    limit: 100,
    order: "desc",
  });
  let orgs = list.data;
  let after = list.listMetadata.after;

  while (after) {
    list = await workos.organizations.listOrganizations({
      limit: 100,
      after: after,
      order: "desc",
    });
    orgs = orgs.concat(list.data);
    after = list.listMetadata.after;
  }

  return orgs;
}

async function deleteOrgs(orgId: string): Promise<void> {
  try {
    await workos.organizations.deleteOrganization(orgId);
    console.log(`Successfully deleted user ${orgId}`);
  } catch (error) {
    if (error instanceof RateLimitExceededException) {
      console.error(`Rate limit exceeded for Org ${orgId}, retrying...`);
      // Wait and retry once
      await new Promise((resolve) => setTimeout(resolve, 1000));
      return deleteOrgs(orgId);
    } else if (error && typeof error === "object" && "errors" in error) {
      const workosError = error as WorkOSResponseError;
      const errorMessage = workosError.errors;
      console.error(`Failed to delete Org ${orgId}: ${errorMessage}`);
    } else {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error";
      console.error(`Unexpected error deleting Org ${orgId}: ${errorMessage}`);
    }
  }
}

export async function processOrgs(): Promise<void> {
  try {
    const orgs = await getWorkOSOrganizations();
    console.log(`Processing ${orgs.length} orgs...`);

    const results = await Promise.allSettled(
      orgs.map((org) => limit(() => deleteOrgs(org.id)))
    );

    const successful = results.filter((r) => r.status === "fulfilled").length;
    const failed = results.filter((r) => r.status === "rejected").length;

    console.log(
      `Completed processing users:\n- Successfully deleted: ${successful}\n- Failed: ${failed}`
    );
  } catch (error) {
    console.error("Failed to process users:", error);
    throw error;
  }
}

processOrgs().then(() => process.exit(0));
