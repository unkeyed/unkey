import * as clack from "@clack/prompts";
import { and, eq, isNull, schema } from "@unkey/db";
import { promptForApiRequestGeneration, promptForBatchSize, withDatabase } from "./batch-helper";
import { insertVerificationEvents } from "./batch-operations";
import { type KeyInfo, generateKeyHash, generateKeyName } from "./event-generator";
import { clickhouse, connectDatabase, generateMetadata, generateRandomString } from "./utils";

const DEFAULT_BATCH_SIZE = 50_000;

/**
 * Fetches existing APIs for the workspace
 */
async function getAPIs(workspaceId: string) {
  return withDatabase(async (db) => {
    return db
      .select({
        id: schema.apis.id,
        name: schema.apis.name,
        keyAuthId: schema.apis.keyAuthId,
      })
      .from(schema.apis)
      .where(and(eq(schema.apis.workspaceId, workspaceId), isNull(schema.apis.deletedAtM)))
      .orderBy(schema.apis.name);
  }, connectDatabase);
}

/**
 * Creates a new API with key authentication
 */
async function createApi(workspaceId: string, name: string) {
  return withDatabase(async (db) => {
    const keyAuthId = `kauth_${generateRandomString(24)}`;
    const payload = {
      id: keyAuthId,
      workspaceId: workspaceId,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      storeEncryptedKeys: Math.random() > 0.7, // 30% chance to store encrypted keys
      defaultPrefix: `uk_${generateRandomString(2)}`,
      defaultBytes: 16,
      sizeApprox: 0,
      sizeLastUpdatedAt: Date.now(),
    };
    await db.insert(schema.keyAuth).values(payload);

    // Then create the API record
    const apiId = `api_${generateRandomString(24)}`;
    const apisPayload = {
      id: apiId,
      name: name,
      workspaceId: workspaceId,
      ipWhitelist: Math.random() > 0.8 ? "192.168.1.1,10.0.0.1" : null, // 20% chance to have IP whitelist
      authType: "key",
      keyAuthId: keyAuthId,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      deletedAt: null,
      deletedBy: null,
    };
    //@ts-expect-error I don't know why authType type doesn't match with match schema but it works
    await db.insert(schema.apis).values(apisPayload);
    return { apiId, keyAuthId };
  }, connectDatabase);
}

/**
 * Create keys for an API with realistic attributes
 */
async function createKeysForApi(
  workspaceId: string,
  keyAuthId: string,
  count: number,
): Promise<KeyInfo[]> {
  return withDatabase(async (db) => {
    const createdKeys: KeyInfo[] = [];

    for (let i = 0; i < count; i++) {
      // Generate key details
      const keyId = `key_${generateRandomString(24)}`;
      const keyPrefix = `uk_${generateRandomString(3)}`;
      const fullKeyContent = `${keyPrefix}_${generateRandomString(32)}`;
      const keyHash = generateKeyHash(fullKeyContent);

      // Determine key attributes with realistic distributions
      const name = generateKeyName();
      const ownerId = `user_${generateRandomString(16)}`;
      const meta = Math.random() > 0.3 ? JSON.stringify(generateMetadata()) : null; // 70% chance to have metadata

      // Expiration: 25% of keys expire within a year
      const expires =
        Math.random() > 0.75
          ? new Date(Date.now() + Math.floor(Math.random() * 365) * 86400000)
          : null;

      // Rate limiting: 40% of keys have rate limits
      const hasRatelimit = Math.random() > 0.6;
      const ratelimitLimit = hasRatelimit ? 100 * (1 + Math.floor(Math.random() * 100)) : null;
      const ratelimitDuration = hasRatelimit
        ? 60 * 1000 * (1 + Math.floor(Math.random() * 60))
        : null;
      const ratelimitAsync = hasRatelimit ? Math.random() > 0.7 : null;

      // Usage limits: 20% of keys have usage limits
      const hasUsageLimit = Math.random() > 0.8;
      const remaining = hasUsageLimit ? 1000 * (1 + Math.floor(Math.random() * 100)) : null;

      // Environment
      const environment =
        Math.random() > 0.6
          ? ["production", "development", "test", "staging"][Math.floor(Math.random() * 4)]
          : null;

      // Enabled/disabled: 95% of keys are enabled
      const enabled = Math.random() > 0.05;

      const payload = {
        id: keyId,
        keyAuthId: keyAuthId,
        hash: keyHash,
        start: keyPrefix,
        workspaceId: workspaceId,
        forWorkspaceId: null,
        name: name,
        ownerId: ownerId,
        identityId: null,
        meta: meta,
        expires: expires,
        createdAt: Date.now(),
        updatedAt: Date.now(),
        deletedAt: null,
        deletedBy: null,
        refillDay: null,
        refillAmount: null,
        lastRefillAt: null,
        enabled: enabled,
        remaining: remaining,
        ratelimitAsync: ratelimitAsync,
        ratelimitLimit: ratelimitLimit,
        ratelimitDuration: ratelimitDuration,
        environment: environment,
      };

      // Insert the key
      await db.insert(schema.keys).values(payload);

      createdKeys.push({
        id: keyId,
        name: name,
        prefix: keyPrefix,
        enabled: enabled,
        hasRatelimit,
        hasUsageLimit,
      });

      // Update the size approximation in keyAuth
      if (i === count - 1) {
        await db
          .update(schema.keyAuth)
          .set({
            sizeApprox: count,
            sizeLastUpdatedAt: Date.now(),
          })
          .where(eq(schema.keyAuth.id, keyAuthId));
      }
    }

    return createdKeys;
  }, connectDatabase);
}

/**
 * Get existing keys for an API/keyspace
 */
async function getExistingKeys(workspaceId: string, keyAuthId: string): Promise<KeyInfo[]> {
  return withDatabase(async (db) => {
    const keys = await db
      .select({
        id: schema.keys.id,
        name: schema.keys.name,
        start: schema.keys.start,
        enabled: schema.keys.enabled,
        ratelimitLimit: schema.keys.ratelimitLimit,
        remaining: schema.keys.remaining,
      })
      .from(schema.keys)
      .where(
        and(
          eq(schema.keys.workspaceId, workspaceId),
          eq(schema.keys.keyAuthId, keyAuthId),
          isNull(schema.keys.deletedAtM),
        ),
      );

    return keys.map((key: any) => ({
      id: key.id,
      name: key.name ?? `Unnamed Key (${key.id})`,
      prefix: key.start,
      enabled: Boolean(key.enabled),
      hasRatelimit: key.ratelimitLimit !== null,
      hasUsageLimit: key.remaining !== null,
    }));
  }, connectDatabase);
}

/**
 * Main function to seed API and keys
 */
export async function seedApiAndKeys(workspaceId: string, count: number) {
  // First, fetch existing APIs
  const existingApis = await getAPIs(workspaceId);

  // API selection logic
  let apiId: string;
  let keyAuthId: string;
  let apiName: string;

  if (existingApis.length > 0) {
    const apiChoice = await clack.select({
      message: "Would you like to use an existing API or create a new one?",
      options: [
        { value: "existing", label: "Use an existing API" },
        { value: "new", label: "Create a new API" },
      ],
    });

    if (clack.isCancel(apiChoice)) {
      clack.cancel("Operation cancelled");
      process.exit(0);
    }

    if (apiChoice === "existing") {
      // Get user to select an existing API
      const selectedApi = await clack.select({
        message: "Select an API:",
        options: existingApis.map((api: any) => ({
          value: JSON.stringify({ id: api.id, keyAuthId: api.keyAuthId }),
          label: api.name,
        })),
      });

      if (clack.isCancel(selectedApi)) {
        clack.cancel("Operation cancelled");
        process.exit(0);
      }

      const { id, keyAuthId: selectedKeyAuthId } = JSON.parse(selectedApi as string);
      apiId = id;
      keyAuthId = selectedKeyAuthId;
      apiName = existingApis.find((api: any) => api.id === apiId)?.name || "Unknown";
    } else {
      // Create a new API
      apiName = (await clack.text({
        message: "Enter a name for the new API:",
        defaultValue: `API ${new Date().toISOString().substring(0, 10)}`,
        validate(value) {
          if (!value || value.trim().length === 0) {
            return "Please enter a valid API name";
          }
        },
      })) as string;

      if (clack.isCancel(apiName)) {
        clack.cancel("Operation cancelled");
        process.exit(0);
      }

      const result = await createApi(workspaceId, apiName as string);
      apiId = result.apiId;
      keyAuthId = result.keyAuthId;
    }
  } else {
    // No existing APIs, create a new one
    apiName = (await clack.text({
      message: "No existing APIs found. Enter a name for the new API:",
      defaultValue: `API ${new Date().toISOString().substring(0, 10)}`,
      validate(value) {
        if (!value || value.trim().length === 0) {
          return "Please enter a valid API name";
        }
      },
    })) as string;

    if (clack.isCancel(apiName)) {
      clack.cancel("Operation cancelled");
      process.exit(0);
    }

    const result = await createApi(workspaceId, apiName as string);
    apiId = result.apiId;
    keyAuthId = result.keyAuthId;
  }

  // Key management
  const existingKeys = await getExistingKeys(workspaceId, keyAuthId);
  let keys: KeyInfo[] = existingKeys;

  // Ask if the user wants to create additional keys if using an existing API with keys
  if (existingKeys.length > 0) {
    const createMoreKeys = await clack.confirm({
      message: `API ${apiName} already has ${existingKeys.length} keys. Would you like to create additional keys?`,
      initialValue: false,
    });

    if (clack.isCancel(createMoreKeys)) {
      clack.cancel("Operation cancelled");
      process.exit(0);
    }

    if (createMoreKeys) {
      // Ask how many additional keys to create
      const keyCount = await clack.text({
        message: "How many additional keys would you like to create for this API?",
        defaultValue: "100",
        validate(value) {
          const num = Number.parseInt(value, 10);
          if (Number.isNaN(num) || num <= 0) {
            return "Please enter a valid positive number";
          }
        },
      });

      if (clack.isCancel(keyCount)) {
        clack.cancel("Operation cancelled");
        process.exit(0);
      }

      const numKeyCount = Number.parseInt(keyCount as string, 10);
      const newKeys = await createKeysForApi(workspaceId, keyAuthId, numKeyCount);
      // Add the new keys to our existing keys array
      keys = [...existingKeys, ...newKeys];
    }
  } else {
    // Either a new API or an existing API with no keys
    // Ask how many keys to create
    const keyCount = await clack.text({
      message: "How many keys would you like to create for this API?",
      defaultValue: "100",
      validate(value) {
        const num = Number.parseInt(value, 10);
        if (Number.isNaN(num) || num <= 0) {
          return "Please enter a valid positive number";
        }
      },
    });

    if (clack.isCancel(keyCount)) {
      clack.cancel("Operation cancelled");
      process.exit(0);
    }

    const numKeyCount = Number.parseInt(keyCount as string, 10);
    keys = await createKeysForApi(workspaceId, keyAuthId, numKeyCount);
  }

  // Configure batch size using utility
  const batchSize = await promptForBatchSize(DEFAULT_BATCH_SIZE);

  // Check if we have keys before offering to generate verification events
  if (keys.length > 0) {
    const generateVerifications = await clack.confirm({
      message: "Would you like to generate key verification events?",
      initialValue: true,
    });

    if (clack.isCancel(generateVerifications)) {
      clack.cancel("Operation cancelled");
      process.exit(0);
    }

    if (generateVerifications) {
      const verificationCount = await clack.text({
        message: "How many verification events would you like to create?",
        defaultValue: count.toString(),
        validate(value) {
          const num = Number.parseInt(value, 10);
          if (Number.isNaN(num) || num <= 0) {
            return "Please enter a valid positive number";
          }
        },
      });

      if (clack.isCancel(verificationCount)) {
        clack.cancel("Operation cancelled");
        process.exit(0);
      }

      const numVerificationCount = Number.parseInt(verificationCount as string, 10);
      const generateApiLogs = await promptForApiRequestGeneration();

      await insertVerificationEvents(
        clickhouse,
        workspaceId,
        keyAuthId,
        keys,
        numVerificationCount,
        generateApiLogs,
        batchSize,
      );
    }
  } else {
    console.info(
      "No keys available for verification events. Skipping verification event generation.",
    );
  }

  return {
    apiId,
    apiName,
    keyAuthId,
    keyCount: keys.length,
    batchSize,
  };
}
