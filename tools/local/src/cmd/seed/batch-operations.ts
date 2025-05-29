import {
  type BatchInsertProgress,
  initBatchInsertProgress,
  updateProgressWithETA,
} from "./batch-helper";
import {
  type KeyInfo,
  biasVerificationOutcome,
  generateMatchingApiRequestForRatelimit,
  generateMatchingApiRequestForVerification,
  generateRatelimitEvent,
  selectKeyWithNormalDistribution,
} from "./event-generator";
import { generateUuid } from "./utils";

/**
 * Inserts verification events in batches with progress tracking
 */
export async function insertVerificationEvents(
  clickhouse: any,
  workspaceId: string,
  keyAuthId: string,
  keys: KeyInfo[],
  count: number,
  generateMatchingApiRequests = true,
  batchSize = 50_000,
) {
  const doVerificationInsert = clickhouse.verifications.insert;
  const doApiInsert = clickhouse.api.insert;

  // Initialize progress tracking
  const progress: BatchInsertProgress = initBatchInsertProgress(count);

  // Sort keys to ensure consistent normal distribution selection
  const sortedKeys = [...keys].sort((a, b) => a.id.localeCompare(b.id));

  // Track usage stats for reporting
  const keyUsageCounter = new Map();
  sortedKeys.forEach((key) => keyUsageCounter.set(key.id, 0));

  try {
    while (progress.insertedCount < count) {
      progress.batchNumber++;
      const currentBatchSize = Math.min(batchSize, count - progress.insertedCount);

      // Update progress
      updateProgressWithETA(progress);

      const batchOfVerificationRecords = [];
      const batchOfApiRequestRecords = [];

      for (let i = 0; i < currentBatchSize; i++) {
        // Select a key using normal distribution algorithm
        const key = selectKeyWithNormalDistribution(sortedKeys);

        // Increment the usage counter for this key
        keyUsageCounter.set(key.id, keyUsageCounter.get(key.id) + 1);

        // For some verification events, create matching API request logs
        const createApiRequestLog = generateMatchingApiRequests && Math.random() < 0.8; // 80% chance
        const requestId = generateUuid();

        // Create the verification event, biasing outcomes based on key properties
        const verificationEvent = biasVerificationOutcome(key, workspaceId, keyAuthId, requestId);
        batchOfVerificationRecords.push(verificationEvent);

        // If needed, create a matching API request
        if (createApiRequestLog) {
          const apiRequest = generateMatchingApiRequestForVerification(
            workspaceId,
            verificationEvent,
            key.prefix,
          );
          batchOfApiRequestRecords.push(apiRequest);
        }
      }

      // Insert verification events
      if (batchOfVerificationRecords.length > 0) {
        await doVerificationInsert(batchOfVerificationRecords);

        // Insert matching API requests if any
        if (batchOfApiRequestRecords.length > 0) {
          await doApiInsert(batchOfApiRequestRecords);
        }
      }

      // Update progress after batch completion
      progress.insertedCount += currentBatchSize;
      updateProgressWithETA(progress);
    }

    process.stdout.write("\n");

    console.info(
      `\n✅ Successfully inserted ${count.toLocaleString()} verification events with ${
        generateMatchingApiRequests ? "matching API requests" : "no matching API requests"
      }.`,
    );

    return { keyUsageStats: Object.fromEntries(keyUsageCounter) };
  } catch (error: unknown) {
    // End progress with a newline
    process.stdout.write("\n");

    console.error(
      `❌ Error inserting data during batch ${progress.batchNumber}: ${(error as { message: string }).message}`,
    );
    console.error("ClickHouse Insert Error Details:", error);
    throw error;
  }
}

/**
 * Inserts ratelimit events in batches with progress tracking
 */
export async function insertRatelimitEvents(
  clickhouse: any,
  workspaceId: string,
  namespaceId: string,
  count: number,
  generateMatchingApiRequests = true,
  batchSize = 50_000,
) {
  const doRatelimitInsert = clickhouse.ratelimits.insert;
  const doApiInsert = clickhouse.api.insert;

  // Initialize progress tracking
  const progress: BatchInsertProgress = initBatchInsertProgress(count);

  try {
    while (progress.insertedCount < count) {
      progress.batchNumber++;
      const currentBatchSize = Math.min(batchSize, count - progress.insertedCount);

      // Update progress display before processing batch
      updateProgressWithETA(progress);

      const batchOfRatelimitRecords = [];
      const batchOfApiRequestRecords = [];

      // Create batch of records
      for (let i = 0; i < currentBatchSize; i++) {
        // For some ratelimit events, we want to create matching API request logs
        const createApiRequestLog = generateMatchingApiRequests && Math.random() < 0.8; // 80% chance
        const requestId = generateUuid();

        // Create the ratelimit event
        const ratelimitEvent = generateRatelimitEvent(workspaceId, namespaceId, requestId);
        batchOfRatelimitRecords.push(ratelimitEvent);

        // If needed, create a matching API request
        if (createApiRequestLog) {
          const apiRequest = generateMatchingApiRequestForRatelimit(workspaceId, ratelimitEvent);
          batchOfApiRequestRecords.push(apiRequest);
        }
      }

      // Insert ratelimit records
      if (batchOfRatelimitRecords.length > 0) {
        await doRatelimitInsert(batchOfRatelimitRecords);

        // Insert API request records if needed
        if (batchOfApiRequestRecords.length > 0) {
          await doApiInsert(batchOfApiRequestRecords);
        }
      }

      // Update progress after batch completion
      progress.insertedCount += currentBatchSize;
      updateProgressWithETA(progress);
    }

    // End progress with a newline
    process.stdout.write("\n");

    const totalTimeElapsed = Date.now() - progress.startTime;
    const avgRecordsPerSecond = count / (totalTimeElapsed / 1000);

    console.info(
      `✅ Successfully inserted ${count.toLocaleString()} ratelimit events with ${
        generateMatchingApiRequests ? "matching API requests" : "no matching API requests"
      } (avg: ${Math.round(avgRecordsPerSecond).toLocaleString()} records/sec)`,
    );

    return { totalTimeElapsed, avgRecordsPerSecond };
  } catch (error: unknown) {
    // End progress with a newline
    process.stdout.write("\n");

    console.error(
      `❌ Error inserting data during batch ${progress.batchNumber}: ${(error as { message: string }).message}`,
    );
    console.error("ClickHouse Insert Error Details:", error);
    throw error;
  }
}
