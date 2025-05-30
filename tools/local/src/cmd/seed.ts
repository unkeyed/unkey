import * as clack from "@clack/prompts";
import { seedApiAndKeys } from "./seed/apis";
import { seedApiRequestLogs } from "./seed/logs";
import { seedRatelimitData } from "./seed/ratelimit";
import { getRecordCount, getWorkspaceOptions, verifyWorkspace } from "./seed/utils";

const DEFAULT_COUNT = 1_000_000;
type SeedingRoute = "Logs" | "APIs" | "Ratelimit";

// Function to prompt user to select a seeding route
async function selectSeedingRoute(): Promise<SeedingRoute> {
  const selected = await clack.select({
    message: "Select which type of data to seed:",
    options: [
      {
        label: "API Request Logs",
        value: "Logs",
        hint: "Seed API request logs data",
      },
      {
        label: "APIs and Keys",
        value: "APIs",
        hint: "Seed API definitions, keys, and verification events",
      },
      {
        label: "Ratelimit",
        value: "Ratelimit",
        hint: "Seed ratelimit data",
      },
    ],
  });

  if (clack.isCancel(selected)) {
    clack.cancel("Operation cancelled");
    process.exit(0);
  }

  return selected as SeedingRoute;
}

export async function seed(options: { ws?: string } = {}) {
  clack.intro("Seeding data for Unkey");

  // Select the seeding route
  const seedingRoute = await selectSeedingRoute();

  let workspaceId = options.ws;
  let workspaceName = "";

  // If workspace ID is provided, verify it exists
  if (workspaceId) {
    const workspace = await verifyWorkspace(workspaceId);
    if (workspace) {
      workspaceName = workspace.name;
      clack.log.success(`Using workspace: ${workspaceName} (${workspaceId})`);
    } else {
      clack.log.warn(`Workspace with ID ${workspaceId} not found`);
      workspaceId = undefined;
    }
  }

  // If no workspace ID or invalid ID, prompt user to select one
  if (!workspaceId) {
    const spinner = clack.spinner();
    spinner.start("Fetching available workspaces");
    const workspaceOptions = await getWorkspaceOptions();
    spinner.stop("Workspaces loaded");

    if (workspaceOptions.length === 0) {
      clack.log.error("No workspaces found in the database");
      clack.cancel("Seeding cancelled");
      return;
    }

    const selected = await clack.select({
      message: "Select a workspace to seed data for",
      options: workspaceOptions,
    });

    if (clack.isCancel(selected)) {
      clack.cancel("Operation cancelled");
      return;
    }

    workspaceId = selected as string;
  }

  let recordCount = DEFAULT_COUNT;
  if (seedingRoute !== "APIs") {
    recordCount = await getRecordCount(DEFAULT_COUNT);
  }

  // Different confirmation message based on the route
  let confirmMessage: string;
  switch (seedingRoute) {
    case "Logs":
      confirmMessage = `Ready to insert ${recordCount} API request logs for workspace ${workspaceId}?`;
      break;
    case "APIs":
      confirmMessage = `Ready to create API, keys, and verification events for workspace ${workspaceId}?`;
      break;
    case "Ratelimit":
      confirmMessage = `Ready to insert ${recordCount} ratelimit records for workspace ${workspaceId}?`;
      break;
    default:
      confirmMessage = `Ready to seed ${seedingRoute} data for workspace ${workspaceId}?`;
  }

  // Confirm seeding with appropriate message
  const confirmed = await clack.confirm({
    message: confirmMessage,
  });

  if (!confirmed || clack.isCancel(confirmed)) {
    clack.cancel("Operation cancelled");
    return;
  }

  try {
    let result: any;
    switch (seedingRoute) {
      case "Logs":
        await seedApiRequestLogs(workspaceId, recordCount);
        clack.log.success(`Inserted ${recordCount} API request logs`);
        break;
      case "APIs":
        result = await seedApiAndKeys(workspaceId, DEFAULT_COUNT);
        clack.log.success(`Created API: ${result.apiName} (${result.apiId})`);
        clack.log.success(`Created ${result.keyCount} keys in keyspace: ${result.keyAuthId}`);
        if (result.verificationCount > 0) {
          clack.log.success(`Inserted ${result.verificationCount} verification events`);
        }
        break;
      case "Ratelimit":
        result = await seedRatelimitData(workspaceId, recordCount);
        clack.log.success(`Created namespace: ${result.namespaceName} (${result.namespaceId})`);
        clack.log.success(`Inserted ${result.eventCount} ratelimit events`);
        break;
    }

    clack.outro(`Seeding of ${seedingRoute} data completed successfully!`);
  } catch (error: any) {
    clack.log.error(`Seeding failed: ${error.message}`);
    clack.outro("Please check your database and ClickHouse connection and table configuration.");
  }
}
