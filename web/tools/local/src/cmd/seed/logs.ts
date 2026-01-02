import {
  type BatchInsertProgress,
  initBatchInsertProgress,
  promptForBatchSize,
  updateProgressWithETA,
} from "./batch-helper";
import { clickhouse, generateRandomApiRequest } from "./utils";

const DEFAULT_BATCH_SIZE = 50_000;

/**
 * Inserts API request logs in batches with progress tracking
 */
async function insertApiRequestLogs(
  clickhouse: any,
  workspaceId: string,
  count: number,
  batchSize = DEFAULT_BATCH_SIZE,
) {
  const doInsert = clickhouse.api.insert;

  // Initialize progress tracking
  const progress: BatchInsertProgress = initBatchInsertProgress(count);

  try {
    while (progress.insertedCount < count) {
      progress.batchNumber++;
      const currentBatchSize = Math.min(batchSize, count - progress.insertedCount);

      // Update progress before processing batch
      updateProgressWithETA(progress);

      const batchOfRecords = [];
      for (let i = 0; i < currentBatchSize; i++) {
        const requestData = generateRandomApiRequest(workspaceId);
        batchOfRecords.push(requestData);
      }

      if (batchOfRecords.length > 0) {
        await doInsert(batchOfRecords);
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
      `✅ Successfully inserted ${count.toLocaleString()} API request logs with realistic 30-day time distribution (avg: ${Math.round(
        avgRecordsPerSecond,
      ).toLocaleString()} records/sec)`,
    );

    return {
      count,
      batchSize,
      workspaceId,
      totalTimeElapsed,
      avgRecordsPerSecond,
    };
  } catch (error: any) {
    // End progress with a newline
    process.stdout.write("\n");

    console.error(`❌ Error inserting data during batch ${progress.batchNumber}: ${error.message}`);
    console.error("ClickHouse Insert Error Details:", error);
    throw error;
  }
}

/**
 * Main function to seed API request logs
 */
export async function seedApiRequestLogs(workspaceId: string, count: number) {
  // Configure batch size using utility
  const batchSize = await promptForBatchSize(DEFAULT_BATCH_SIZE);

  // Insert API request logs
  return await insertApiRequestLogs(clickhouse, workspaceId, count, batchSize);
}
