import crypto from "node:crypto";
import * as clack from "@clack/prompts";
import { ClickHouse } from "@unkey/clickhouse";
import { eq, mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";

// Generate UUID using crypto
function generateUuid() {
  return crypto.randomUUID();
}

// Environment function
function env() {
  return {
    CLICKHOUSE_URL: "http://default:password@localhost:8123",
    DB_URL: "mysql://unkey:password@localhost:3306/unkey",
  };
}

export const clickhouse = new ClickHouse({ url: env().CLICKHOUSE_URL });

// Connect to the database
async function connectDatabase() {
  console.log("Connecting to database...");
  let err: Error | undefined = undefined;
  for (let i = 1; i <= 10; i++) {
    try {
      const conn = await mysql.createConnection(env().DB_URL);
      console.log("Pinging database...");
      await conn.ping();
      console.log("Connected to database");
      return { db: mysqlDrizzle(conn, { schema, mode: "default" }), conn };
    } catch (e) {
      err = e as Error;
      console.log(`Connection attempt ${i} failed, retrying in ${i}s...`);
      await new Promise((r) => setTimeout(r, 1000 * i));
    }
  }
  throw err;
}

// List available workspaces for selection
async function getWorkspaceOptions() {
  const { db, conn } = await connectDatabase();

  try {
    // Get all workspaces with their names
    const workspaces = await db
      .select({
        id: schema.workspaces.id,
        name: schema.workspaces.name,
      })
      .from(schema.workspaces)
      .limit(20);

    return workspaces.map((w) => ({
      value: w.id,
      label: `${w.name}`,
      hint: w.id,
    }));
  } finally {
    // Close the database connection
    await conn.end();
  }
}

// Verify if a workspace exists
async function verifyWorkspace(workspaceId: string) {
  const { db, conn } = await connectDatabase();

  try {
    const workspace = await db
      .select({
        id: schema.workspaces.id,
        name: schema.workspaces.name,
      })
      .from(schema.workspaces)
      .where(eq(schema.workspaces.id, workspaceId))
      .limit(1);

    return workspace.length > 0 ? workspace[0] : null;
  } finally {
    await conn.end();
  }
}

// Generate random API request data
function generateRandomApiRequest(workspaceId: string) {
  const methods = ["GET", "POST", "PUT", "DELETE", "PATCH"];
  const paths = [
    "/v1/keys.create",
    "/v1/keys.verify",
    "/v1/keys.delete",
    "/v1/keys.update",
    "/v1/apis.list",
  ];
  const statusCodes = [200, 201, 400, 401, 403, 404, 500];
  const userAgents = [
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15",
    "PostmanRuntime/7.29.0",
    "curl/7.68.0",
    "Node.js/16.13.2 (axios/0.26.1)",
  ];

  return {
    request_id: generateUuid(),
    time: Date.now(),
    workspace_id: workspaceId,
    host: "api.unkey.dev",
    method: methods[Math.floor(Math.random() * methods.length)],
    path: paths[Math.floor(Math.random() * paths.length)],
    request_headers: ["content-type: application/json", "authorization: Bearer test"],
    request_body: JSON.stringify({ test: "data" }),
    response_status: statusCodes[Math.floor(Math.random() * statusCodes.length)],
    response_headers: ["content-type: application/json"],
    response_body: JSON.stringify({ success: true }),
    error: Math.random() > 0.8 ? "Error processing request" : "",
    service_latency: Math.floor(Math.random() * 500),
    user_agent: userAgents[Math.floor(Math.random() * userAgents.length)],
    ip_address: `192.168.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`,
    continent:
      Math.random() > 0.2
        ? ["North America", "Europe", "Asia", "Australia", "South America", "Africa"][
            Math.floor(Math.random() * 6)
          ]
        : "",
    city:
      Math.random() > 0.2
        ? ["New York", "London", "Toronto", "Berlin", "Paris", "Tokyo", "Sydney"][
            Math.floor(Math.random() * 7)
          ]
        : "",
    country:
      Math.random() > 0.2
        ? ["US", "UK", "CA", "DE", "FR", "JP", "AU", "BR", "IN"][Math.floor(Math.random() * 9)]
        : "",
    colo: Math.random() > 0.2 ? `colo-${Math.floor(Math.random() * 10)}` : "",
  };
}

// Insert data into Clickhouse
async function insertClickhouseData(workspaceId: string, count: number) {
  try {
    const spinner = clack.spinner();
    spinner.start(`Inserting ${count} records for workspace ${workspaceId}`);

    // Get the insert function from clickhouse
    const insertRequest = clickhouse.api.insert;

    // Insert multiple records
    for (let i = 0; i < count; i++) {
      const requestData = generateRandomApiRequest(workspaceId);
      await insertRequest(requestData);

      // Update spinner every 100 records
      if ((i + 1) % 100 === 0) {
        spinner.message(`Inserted ${i + 1}/${count} records`);
      }
    }

    spinner.stop(`Successfully inserted ${count} records into Clickhouse`);
  } catch (error) {
    clack.log.error(`Error inserting data: ${error}`);
    throw error;
  }
}

// Get the count of records to insert
async function getRecordCount(defaultCount = 1000) {
  const count = await clack.text({
    message: "How many records would you like to insert?",
    defaultValue: defaultCount.toString(),
    validate(value) {
      const num = Number.parseInt(value, 10);
      if (Number.isNaN(num) || num <= 0) {
        return "Please enter a valid positive number";
      }
    },
  });

  if (clack.isCancel(count)) {
    clack.cancel("Operation cancelled");
    process.exit(0);
  }

  return Number.parseInt(count as string, 10);
}

// Main seed function
export async function seed(options: { ws?: string } = {}) {
  clack.intro("Seeding Clickhouse with random API request data");

  let workspaceId = options.ws;
  let workspaceName = "";

  // If workspace ID is provided, verify it exists
  if (workspaceId) {
    const workspace = await verifyWorkspace(workspaceId);
    if (!workspace) {
      clack.log.warn(`Workspace with ID ${workspaceId} not found`);
      workspaceId = undefined;
    } else {
      workspaceName = workspace.name;
      clack.log.success(`Using workspace: ${workspaceName} (${workspaceId})`);
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

  // Get number of records to insert
  const recordCount = await getRecordCount();

  // Confirm seeding
  const confirmed = await clack.confirm({
    message: `Ready to insert ${recordCount} records for workspace ${workspaceId}?`,
  });

  if (!confirmed || clack.isCancel(confirmed)) {
    clack.cancel("Operation cancelled");
    return;
  }

  // Insert data
  await insertClickhouseData(workspaceId, recordCount);

  clack.outro("Seeding completed successfully!");
}
