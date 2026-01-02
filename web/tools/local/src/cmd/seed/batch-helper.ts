import * as clack from "@clack/prompts";
import type mysql from "mysql2/promise";
import { type DrizzleReturnType, createProgressBar, formatDuration } from "./utils";

export type BatchInsertionOptions = {
  count: number;
  batchSize?: number;
  defaultBatchSize?: number;
  maxBatchSize?: number;
  generateMatchingApiRequests?: boolean;
};

export type BatchInsertProgress = {
  insertedCount: number;
  totalCount: number;
  batchNumber: number;
  startTime: number;
  lastProgressUpdate: number;
};

/**
 * Initializes batch insertion progress tracking
 */
export function initBatchInsertProgress(totalCount: number): BatchInsertProgress {
  return {
    insertedCount: 0,
    totalCount,
    batchNumber: 0,
    startTime: Date.now(),
    lastProgressUpdate: 0,
  };
}

/**
 * Updates progress display with ETA calculation
 */
export function updateProgressWithETA(progress: BatchInsertProgress) {
  // Skip if no progress yet
  if (progress.insertedCount === 0) {
    return;
  }

  const currentTime = Date.now();

  // Only update every 2 seconds to avoid too many log lines
  if (
    currentTime - progress.lastProgressUpdate < 2000 &&
    progress.insertedCount < progress.totalCount
  ) {
    return;
  }

  progress.lastProgressUpdate = currentTime;
  const elapsedTime = currentTime - progress.startTime;
  const percentComplete = (progress.insertedCount / progress.totalCount) * 100;

  // Calculate records per second
  const recordsPerSecond = progress.insertedCount / (elapsedTime / 1000);

  // Calculate estimated time remaining
  const remainingRecords = progress.totalCount - progress.insertedCount;
  const estimatedSecondsRemaining = recordsPerSecond > 0 ? remainingRecords / recordsPerSecond : 0;

  // Format progress message
  const progressBar = createProgressBar(percentComplete);
  const message =
    `Progress: ${progressBar} ${percentComplete.toFixed(1)}% | ` +
    `${Math.round(recordsPerSecond).toLocaleString()}/sec | ` +
    `ETA: ${formatDuration(estimatedSecondsRemaining * 1000)} | ` +
    `Batch #${
      progress.batchNumber
    } | ${progress.insertedCount.toLocaleString()}/${progress.totalCount.toLocaleString()} records`;

  process.stdout.write(`\r\x1b[K${message}`);
}

/**
 * Handles customization of batch size through interactive prompts
 */
export async function promptForBatchSize(
  defaultBatchSize = 50_000,
  maxBatchSize = 1_000_000,
): Promise<number> {
  const customizeBatchSize = await clack.confirm({
    message: "Would you like to customize the batch size?",
    initialValue: false,
  });

  if (clack.isCancel(customizeBatchSize)) {
    clack.cancel("Operation cancelled");
    process.exit(0);
  }

  if (!customizeBatchSize) {
    return defaultBatchSize;
  }

  const userBatchSize = await clack.text({
    message: `Enter batch size (default: ${defaultBatchSize.toLocaleString()}):`,
    defaultValue: defaultBatchSize.toString(),
    validate(value) {
      const num = Number.parseInt(value, 10);
      if (Number.isNaN(num) || num <= 0) {
        return "Please enter a valid positive number";
      }
      if (num > maxBatchSize) {
        return `Batch size too large (max: ${maxBatchSize.toLocaleString()})`;
      }
    },
  });

  if (clack.isCancel(userBatchSize)) {
    clack.cancel("Operation cancelled");
    process.exit(0);
  }

  return Number.parseInt(userBatchSize as string, 10);
}

/**
 * Prompts user to confirm whether to generate matching API requests
 */
export async function promptForApiRequestGeneration(): Promise<boolean> {
  const generateApiLogs = await clack.confirm({
    message: "Would you like to generate matching API request logs?",
    initialValue: true,
  });

  if (clack.isCancel(generateApiLogs)) {
    clack.cancel("Operation cancelled");
    process.exit(0);
  }

  return generateApiLogs as boolean;
}

/**
 * Generic database operation wrapper
 */

export async function withDatabase<
  TSchema extends Record<string, unknown> = Record<string, unknown>,
>(
  operation: (db: DrizzleReturnType<TSchema>, conn: mysql.Connection) => Promise<unknown>,
  getConnection: () => Promise<{
    db: DrizzleReturnType<TSchema>;
    conn: mysql.Connection;
  }>,
): Promise<unknown> {
  const { db, conn } = await getConnection();
  try {
    return await operation(db, conn);
  } finally {
    await conn.end();
  }
}

/**
 * Generic resource selection helper
 */
export async function selectOrCreateResource<T extends { id: string; name: string }>(
  resourceType: string,
  existingResources: T[],
  createNewFn: (name: string) => Promise<string>,
  defaultName = `${resourceType} ${new Date().toISOString().substring(0, 10)}`,
): Promise<{ id: string; name: string; isNew: boolean }> {
  // If resources exist, ask if user wants to use existing or create new
  if (existingResources.length > 0) {
    const choice = await clack.select({
      message: `Would you like to use an existing ${resourceType} or create a new one?`,
      options: [
        { value: "existing", label: `Use an existing ${resourceType}` },
        { value: "new", label: `Create a new ${resourceType}` },
      ],
    });

    if (clack.isCancel(choice)) {
      clack.cancel("Operation cancelled");
      process.exit(0);
    }

    if (choice === "existing") {
      // Get user to select an existing resource
      const selected = await clack.select({
        message: `Select a ${resourceType}:`,
        options: existingResources.map((resource) => ({
          value: resource.id,
          label: resource.name,
        })),
      });

      if (clack.isCancel(selected)) {
        clack.cancel("Operation cancelled");
        process.exit(0);
      }

      const id = selected as string;
      const name = existingResources.find((r) => r.id === id)?.name || "Unknown";
      return { id, name, isNew: false };
    }
  }

  // Either user chose to create new, or no existing resources
  const namePrompt =
    existingResources.length > 0
      ? `Enter a name for the new ${resourceType}:`
      : `No existing ${resourceType}s found. Enter a name for the new ${resourceType}:`;

  const name = await clack.text({
    message: namePrompt,
    defaultValue: defaultName,
    validate(value) {
      if (!value || value.trim().length === 0) {
        return `Please enter a valid ${resourceType} name`;
      }
    },
  });

  if (clack.isCancel(name)) {
    clack.cancel("Operation cancelled");
    process.exit(0);
  }

  const id = await createNewFn(name as string);
  return { id, name: name as string, isNew: true };
}
