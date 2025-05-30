import * as clack from "@clack/prompts";
import { ClickHouse } from "@unkey/clickhouse";
import { eq, mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";

export function generateUuid() {
  return crypto.randomUUID();
}

function env() {
  return {
    CLICKHOUSE_URL: "http://default:password@localhost:8123",
    DB_URL: "mysql://unkey:password@localhost:3306/unkey",
  };
}

export const clickhouse = new ClickHouse({ url: env().CLICKHOUSE_URL });
export type DrizzleReturnType<T extends Record<string, unknown>> = ReturnType<
  typeof mysqlDrizzle<T>
>;

export async function connectDatabase<
  TSchema extends Record<string, unknown> = Record<string, unknown>,
>(): Promise<{
  db: DrizzleReturnType<TSchema>;
  conn: mysql.Connection;
}> {
  let err: Error | undefined = undefined;
  for (let i = 1; i <= 10; i++) {
    try {
      const conn = await mysql.createConnection(env().DB_URL);
      await conn.ping();
      return {
        db: mysqlDrizzle<TSchema>(conn, {
          schema: schema as unknown as TSchema,
          mode: "default",
        }),
        conn,
      };
    } catch (e) {
      err = e as Error;
      await new Promise((r) => setTimeout(r, 1000 * i));
    }
  }
  throw err;
}

// List available workspaces for selection
export async function getWorkspaceOptions() {
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

    return workspaces.map((w: { id: string; name: string }) => ({
      value: w.id,
      label: `${w.name}`,
      hint: w.id,
    }));
  } finally {
    await conn.end();
  }
}

// Verify if a workspace exists
export async function verifyWorkspace(workspaceId: string) {
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

// Get the count of records to insert
export async function getRecordCount(defaultCount = 100000) {
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

export function generateRandomString(length: number) {
  const charset = "abcdefghijklmnopqrstuvwxyz0123456789";
  let result = "";
  for (let i = 0; i < length; i++) {
    result += charset.charAt(Math.floor(Math.random() * charset.length));
  }
  return result;
}

export function generateMetadata() {
  const metaFields = [
    "environment",
    "region",
    "service",
    "version",
    "tenant",
    "plan",
    "features",
    "limits",
    "tier",
    "customerId",
    "billingId",
    "projectId",
    "tags",
    "labels",
    "origin",
  ];

  // Select a random number of fields to include
  const fieldCount = 1 + Math.floor(Math.random() * 8); // 1-8 fields
  const selectedFields = new Set();

  while (selectedFields.size < fieldCount) {
    const field = metaFields[Math.floor(Math.random() * metaFields.length)];
    selectedFields.add(field);
  }

  const metadata: Record<string, unknown> = {};

  selectedFields.forEach((field) => {
    switch (field) {
      case "environment":
        metadata[field] = ["development", "staging", "production", "test"][
          Math.floor(Math.random() * 4)
        ];
        break;
      case "region":
        metadata[field] = ["us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1", "sa-east-1"][
          Math.floor(Math.random() * 5)
        ];
        break;
      case "service":
        metadata[field] = ["auth", "payments", "analytics", "storage", "compute"][
          Math.floor(Math.random() * 5)
        ];
        break;
      case "version":
        metadata[field] = `v${1 + Math.floor(Math.random() * 3)}.${Math.floor(
          Math.random() * 10,
        )}.${Math.floor(Math.random() * 20)}`;
        break;
      case "tenant":
        metadata[field] = `tenant-${100 + Math.floor(Math.random() * 900)}`;
        break;
      case "plan":
        metadata[field] = ["free", "basic", "pro", "enterprise"][Math.floor(Math.random() * 4)];
        break;
      case "features":
        metadata[field] = {
          rateLimit: Boolean(Math.round(Math.random())),
          analytics: Boolean(Math.round(Math.random())),
          webhooks: Boolean(Math.round(Math.random())),
          multiRegion: Boolean(Math.round(Math.random())),
          customDomains: Boolean(Math.round(Math.random())),
        };
        break;
      case "limits":
        metadata[field] = {
          requestsPerDay: Math.floor(Math.random() * 10000),
          keysPerWorkspace: 10 + Math.floor(Math.random() * 90),
          usersPerOrg: 5 + Math.floor(Math.random() * 45),
        };
        break;
      case "tier":
        metadata[field] = Math.floor(Math.random() * 4);
        break;
      case "customerId":
        metadata[field] = `cus_${generateRandomString(14)}`;
        break;
      case "billingId":
        metadata[field] = `bill_${generateRandomString(14)}`;
        break;
      case "projectId":
        metadata[field] = `proj_${generateRandomString(10)}`;
        break;
      case "tags":
        metadata[field] = Array.from(
          { length: 1 + Math.floor(Math.random() * 4) },
          () =>
            [
              "api",
              "prod",
              "dev",
              "internal",
              "external",
              "test",
              "legacy",
              "beta",
              "alpha",
              "stable",
            ][Math.floor(Math.random() * 10)],
        );
        break;
      case "labels":
        metadata[field] = {
          team: ["frontend", "backend", "devops", "platform", "data"][
            Math.floor(Math.random() * 5)
          ],
          priority: ["low", "medium", "high", "critical"][Math.floor(Math.random() * 4)],
          status: ["active", "deprecated", "experimental"][Math.floor(Math.random() * 3)],
        };
        break;
      case "origin":
        metadata[field] = ["cli", "dashboard", "api", "sdk", "integration"][
          Math.floor(Math.random() * 5)
        ];
        break;
    }
  });

  return metadata;
}

export function generateTimestamp() {
  const now = new Date();
  const thirtyDaysAgo = new Date(now);
  thirtyDaysAgo.setDate(now.getDate() - 30);

  // Define specific spike days with extremely high traffic
  // We'll create major spikes on days 7, 14, 21 ago, plus a few random high-volume days
  const majorSpikeDays = [7, 14, 21]; // Days ago
  const randomSpikeDays = [3, 10, 17, 25, 29]; // Additional spike days

  // Special dates that might have meaning (you can customize these)
  const specialDates = [
    new Date(2025, 2, 15), // March 15, 2025 - hypothetical product launch
    new Date(2025, 3, 1), // April 1, 2025 - promotion day
  ];

  // 40% chance to generate a timestamp on a major spike day
  if (Math.random() < 0.4) {
    // Pick one of the major spike days
    const randomSpikeIndex = Math.floor(Math.random() * majorSpikeDays.length);
    const daysAgo = majorSpikeDays[randomSpikeIndex];

    const spikeDate = new Date(now);
    spikeDate.setDate(now.getDate() - daysAgo);

    // Set to business hours for this spike (8am-6pm)
    const businessHour = 8 + Math.floor(Math.random() * 10);
    spikeDate.setHours(
      businessHour,
      Math.floor(Math.random() * 60),
      Math.floor(Math.random() * 60),
      0,
    );

    return Math.floor(spikeDate.getTime());
  }

  // 20% chance to generate a timestamp on a random spike day
  if (Math.random() < 0.2) {
    const randomSpikeIndex = Math.floor(Math.random() * randomSpikeDays.length);
    const daysAgo = randomSpikeDays[randomSpikeIndex];

    const spikeDate = new Date(now);
    spikeDate.setDate(now.getDate() - daysAgo);

    // Random hour for these secondary spikes
    spikeDate.setHours(
      Math.floor(Math.random() * 24),
      Math.floor(Math.random() * 60),
      Math.floor(Math.random() * 60),
      0,
    );

    return Math.floor(spikeDate.getTime());
  }

  // 10% chance to generate a timestamp on a special date
  if (Math.random() < 0.1 && specialDates.length > 0) {
    const specialDate = specialDates[Math.floor(Math.random() * specialDates.length)];

    // Set to business hours for special dates
    specialDate.setHours(
      9 + Math.floor(Math.random() * 8),
      Math.floor(Math.random() * 60),
      Math.floor(Math.random() * 60),
      0,
    );

    return Math.floor(specialDate.getTime());
  }

  // For the remaining 30% of timestamps, use the weighted distribution
  // This ensures we still have some background traffic on non-spike days
  for (let attempt = 0; attempt < 5; attempt++) {
    // Pick a random time within the 30-day range
    const timestamp = Math.floor(
      thirtyDaysAgo.getTime() + Math.random() * (now.getTime() - thirtyDaysAgo.getTime()),
    );
    const date = new Date(timestamp);

    // Calculate various factors
    const dayOfWeek = date.getDay();
    const hour = date.getHours();

    // Heavily reduce weekend traffic to make weekday spikes more dramatic
    if ((dayOfWeek === 0 || dayOfWeek === 6) && Math.random() < 0.8) {
      continue; // Skip 80% of weekend timestamps
    }

    // Favor business hours
    const isBusinessHours = hour >= 9 && hour <= 17;
    if (!isBusinessHours && Math.random() < 0.7) {
      continue; // Skip 70% of non-business hours
    }

    return timestamp;
  }

  // Fallback - just return a random timestamp within range
  const timestamp = Math.floor(
    thirtyDaysAgo.getTime() + Math.random() * (now.getTime() - thirtyDaysAgo.getTime()),
  );
  return timestamp;
}

export function generateRandomApiRequest(workspaceId: string) {
  const methods = ["GET", "POST", "PUT", "DELETE", "PATCH"];
  const paths = [
    "/v1/keys.create",
    "/v1/keys.verify",
    "/v1/keys.delete",
    "/v1/keys.update",
    "/v1/apis.list",
  ];
  const userAgents = [
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36 Edg/96.0.1054.62",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.1 Safari/605.1.15",
    "PostmanRuntime/7.29.0",
    "curl/7.79.1",
    "Node.js/16.13.2 (axios/0.26.1)",
    "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 15_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.2 Mobile/15E148 Safari/604.1",
  ];

  // Path distribution with verifying keys being most common
  const pathDistribution = Math.random();
  let path: string;
  if (pathDistribution < 0.6) {
    path = "/v1/keys.verify"; // 60% of requests
  } else if (pathDistribution < 0.75) {
    path = "/v1/keys.create"; // 15% of requests
  } else {
    path = paths[Math.floor(Math.random() * paths.length)]; // 25% other endpoints
  }

  // Status code distribution - success is more common
  let responseStatus: number;
  const statusRand = Math.random();
  if (statusRand < 0.85) {
    responseStatus = 200; // 85% success
  } else if (statusRand < 0.9) {
    responseStatus = 401; // 5% unauthorized
  } else if (statusRand < 0.95) {
    responseStatus = 404; // 5% not found
  } else {
    responseStatus = 500; // 5% server error
  }

  // Generate appropriate request and response bodies based on the path and status
  let requestBody: Record<string, unknown> = {};
  let responseBody: Record<string, any> = {};
  let error = "";

  // Generate a key ID and API ID
  const keyId = generateApiKeyId();
  const apiId = generateApiId();

  // Add metadata to most requests
  const includeMeta = Math.random() < 0.7;
  const meta = includeMeta ? generateMetadata() : undefined;

  switch (path) {
    case "/v1/keys.create":
      requestBody = {
        apiId: apiId,
        name: `Key for ${
          ["Production", "Development", "Staging", "Testing"][Math.floor(Math.random() * 4)]
        }`,
        ownerId: `user_${generateRandomString(16)}`,
        expires: Math.random() < 0.3 ? Date.now() + 86400000 * 30 : undefined,
        ratelimit:
          Math.random() < 0.4
            ? {
                type: ["consistent", "sliding"][Math.floor(Math.random() * 2)],
                limit: 100 * Math.floor(1 + Math.random() * 10),
                refillRate: 10 * Math.floor(1 + Math.random() * 10),
                refillInterval: 60 * Math.floor(1 + Math.random() * 10),
              }
            : undefined,
        meta: meta,
      };

      if (responseStatus === 200) {
        responseBody = {
          key: {
            id: keyId,
            apiId: apiId,
            prefix: keyId.substring(0, 8),
            name: requestBody.name,
            ownerId: requestBody.ownerId,
            createdAt: Date.now(),
            expires: requestBody.expires,
            meta: requestBody.meta,
          },
        };
      } else {
        error =
          responseStatus === 401
            ? "Unauthorized"
            : responseStatus === 404
              ? "API not found"
              : "Internal server error";

        responseBody = {
          error: {
            code: String(responseStatus),
            message: error,
          },
        };
      }
      break;

    case "/v1/keys.verify":
      requestBody = {
        key: `${keyId.substring(0, 8)}.${generateRandomString(32)}`,
        apiId: apiId,
      };

      if (responseStatus === 200) {
        responseBody = {
          valid: true,
          keyId: keyId,
          apiId: apiId,
          ownerId: `user_${generateRandomString(16)}`,
          meta: meta,
          ratelimit:
            Math.random() < 0.3
              ? {
                  limit: 1000,
                  remaining: Math.floor(Math.random() * 1000),
                  reset: Date.now() + 3600000,
                }
              : undefined,
        };
      } else {
        error =
          responseStatus === 401
            ? "Invalid API key"
            : responseStatus === 404
              ? "API not found"
              : "Internal server error";

        responseBody = {
          valid: false,
          code: String(responseStatus),
          message: error,
        };
      }
      break;

    case "/v1/keys.delete":
      requestBody = {
        keyId: keyId,
      };

      if (responseStatus === 200) {
        responseBody = {
          success: true,
          keyId: keyId,
        };
      } else {
        error =
          responseStatus === 401
            ? "Unauthorized"
            : responseStatus === 404
              ? "Key not found"
              : "Internal server error";

        responseBody = {
          error: {
            code: String(responseStatus),
            message: error,
          },
        };
      }
      break;

    case "/v1/keys.update":
      requestBody = {
        keyId: keyId,
        name: `Updated key for ${
          ["Production", "Development", "Staging", "Testing"][Math.floor(Math.random() * 4)]
        }`,
        expires: Math.random() < 0.3 ? Date.now() + 86400000 * 60 : undefined,
        meta: meta,
      };

      if (responseStatus === 200) {
        responseBody = {
          key: {
            id: keyId,
            apiId: apiId,
            name: requestBody.name,
            ownerId: `user_${generateRandomString(16)}`,
            updatedAt: Date.now(),
            expires: requestBody.expires,
            meta: requestBody.meta,
          },
        };
      } else {
        error =
          responseStatus === 401
            ? "Unauthorized"
            : responseStatus === 404
              ? "Key not found"
              : "Internal server error";

        responseBody = {
          error: {
            code: String(responseStatus),
            message: error,
          },
        };
      }
      break;

    case "/v1/apis.list":
      requestBody = {
        workspaceId: workspaceId,
        limit: 10,
        offset: Math.floor(Math.random() * 5) * 10,
      };

      if (responseStatus === 200) {
        const apis = [];
        const apiCount = 1 + Math.floor(Math.random() * 7); // 1-7 APIs

        for (let i = 0; i < apiCount; i++) {
          apis.push({
            id: `api_${generateRandomString(24)}`,
            name: `API ${i + 1} for ${
              ["Payments", "Auth", "Data", "Analytics", "Storage"][Math.floor(Math.random() * 5)]
            }`,
            workspaceId: workspaceId,
            createdAt: Date.now() - Math.floor(Math.random() * 30) * 86400000,
            updatedAt: Date.now() - Math.floor(Math.random() * 10) * 86400000,
            keyCount: Math.floor(Math.random() * 50),
            active: Math.random() < 0.9, // 90% active
            meta: Math.random() < 0.5 ? generateMetadata() : undefined,
          });
        }

        responseBody = {
          apis: apis,
          totalCount: 10 + Math.floor(Math.random() * 40),
          cursor: Math.random() < 0.7 ? generateRandomString(20) : undefined,
        };
      } else {
        error = responseStatus === 401 ? "Unauthorized" : "Internal server error";

        responseBody = {
          error: {
            code: String(responseStatus),
            message: error,
          },
        };
      }
      break;

    default:
      requestBody = {
        data: "Generic request",
      };

      if (responseStatus === 200) {
        responseBody = {
          success: true,
        };
      } else {
        error = "Request failed";
        responseBody = {
          error: {
            code: String(responseStatus),
            message: error,
          },
        };
      }
  }

  // Add trace ID and request timing to all responses
  if (responseStatus === 200) {
    responseBody.traceId = `trace_${generateRandomString(16)}`;
    responseBody.meta = responseBody.meta || {};
    responseBody.meta.requestTime = Math.floor(Math.random() * 500);
    responseBody.meta.region = ["us-east-1", "us-west-2", "eu-west-1"][
      Math.floor(Math.random() * 3)
    ];
  }

  const time = generateTimestamp();

  return {
    request_id: generateUuid() as string,
    time,
    workspace_id: workspaceId,
    host: "api.unkey.dev",
    method: methods[Math.floor(Math.random() * methods.length)],
    path: path,
    request_headers: [
      "content-type: application/json",
      `authorization: Bearer test_${generateRandomString(24)}`,
      `x-client-id: ${generateRandomString(12)}`,
      `user-agent: ${userAgents[Math.floor(Math.random() * userAgents.length)]}`,
      "accept: application/json",
      `x-request-id: ${generateRandomString(16)}`,
    ],
    request_body: JSON.stringify(requestBody),
    response_status: responseStatus,
    response_headers: [
      "content-type: application/json",
      "x-ratelimit-limit: 1000",
      `x-ratelimit-remaining: ${Math.floor(Math.random() * 1000)}`,
      `x-request-id: ${generateRandomString(16)}`,
      `x-runtime: ${(Math.random() * 0.5).toFixed(4)}`,
    ],
    response_body: JSON.stringify(responseBody),
    error: error,
    service_latency: Math.floor(Math.random() * 500),
    user_agent: userAgents[Math.floor(Math.random() * userAgents.length)],
    ip_address: generateRealisticIp(),
    continent:
      Math.random() > 0.2
        ? ["North America", "Europe", "Asia", "Australia", "South America", "Africa"][
            Math.floor(Math.random() * 6)
          ]
        : "",
    city:
      Math.random() > 0.2
        ? [
            "New York",
            "London",
            "Toronto",
            "Berlin",
            "Paris",
            "Tokyo",
            "Sydney",
            "Singapore",
            "San Francisco",
            "Amsterdam",
          ][Math.floor(Math.random() * 10)]
        : "",
    country:
      Math.random() > 0.2
        ? ["US", "UK", "CA", "DE", "FR", "JP", "AU", "BR", "IN", "SG", "NL"][
            Math.floor(Math.random() * 11)
          ]
        : "",
    colo: Math.random() > 0.2 ? `colo-${Math.floor(Math.random() * 10)}` : "",
    meta: meta,
  };
}

// Generate a random API key identifier
function generateApiKeyId() {
  return `key_${generateRandomString(24)}`;
}

// Generate a random API identifier
function generateApiId() {
  return `api_${generateRandomString(24)}`;
}

// Generate random IP address that looks realistic
function generateRealisticIp() {
  // Prefer common IP address patterns
  if (Math.random() < 0.7) {
    // Common ranges
    const ranges = [
      [192, 168, null, null], // Private network
      [10, null, null, null], // Private network
      [172, 16, null, null], // Private network
      [34, 100, null, null], // Public range
      [52, 86, null, null], // AWS range
      [13, 107, null, null], // Microsoft range
      [104, 196, null, null], // Google range
    ];

    const range = ranges[Math.floor(Math.random() * ranges.length)];

    return range
      .map((segment) => (segment !== null ? segment : Math.floor(Math.random() * 256)))
      .join(".");
  }
  // Completely random IP
  return `${Math.floor(Math.random() * 256)}.${Math.floor(
    Math.random() * 256,
  )}.${Math.floor(Math.random() * 256)}.${Math.floor(Math.random() * 256)}`;
}

/**
 * Implements the Box-Muller transform to generate normally distributed random numbers
 * Returns a random number with mean 0 and standard deviation 1
 */
function boxMullerTransform() {
  const u1 = Math.random();
  const u2 = Math.random();

  const z0 = Math.sqrt(-2.0 * Math.log(u1)) * Math.cos(2.0 * Math.PI * u2);

  // We only need one value
  return z0;
}

/**
 * Generate a normally distributed random index within a range
 */
export function getNormallyDistributedIndex(
  mean: number,
  stdDev: number,
  min: number,
  max: number,
) {
  let index: number;
  do {
    // Generate normally distributed value
    const normalRandom = boxMullerTransform();

    // Scale to our desired range
    index = Math.round(mean + normalRandom * stdDev);
  } while (index < min || index >= max);

  return index;
}

const POOL_SIZE = 100;
const COMMON_PREFIXES = ["ip:", "user:", "tenant:", "org:", "key:"];

// Initialize pools for different identifier types
export const identifierPools: Record<string, string[]> = {};

// Generate pools of identifiers
export function initializeIdentifierPools() {
  for (const prefix of COMMON_PREFIXES) {
    identifierPools[prefix] = [];
    for (let i = 0; i < POOL_SIZE; i++) {
      identifierPools[prefix].push(`${prefix}${generateRandomString(12)}`);
    }
  }
}

// Select an identifier from a pool using normal distribution
export function selectIdentifier() {
  // First choose a prefix type (this could also be weighted if needed)
  const prefix = COMMON_PREFIXES[Math.floor(Math.random() * COMMON_PREFIXES.length)];

  // Ensure the pool exists
  if (!identifierPools[prefix]) {
    initializeIdentifierPools();
  }

  // Use normal distribution to select from the pool
  // Use mean at 20% of the pool to make some ids "hotter" than others
  const mean = Math.floor(POOL_SIZE * 0.2);
  const stdDev = POOL_SIZE / 5; // Standard deviation of 20% of the pool size
  const index = getNormallyDistributedIndex(mean, stdDev, 0, POOL_SIZE);

  return identifierPools[prefix][index];
}

export const createProgressBar = (percent: number) => {
  const width = 20;
  const filled = Math.floor(width * (percent / 100));
  const empty = width - filled;
  return `[${"█".repeat(filled)}${"·".repeat(empty)}]`;
};

export const formatDuration = (milliseconds: number) => {
  const totalSeconds = Math.floor(milliseconds / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
};
