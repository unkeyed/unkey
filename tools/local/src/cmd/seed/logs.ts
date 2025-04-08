import * as clack from "@clack/prompts";
import { clickhouse, generateRandomApiRequest } from "./utils";

const BATCH_SIZE = 50_000;
export async function seedApiRequestLogs(workspaceId: string, count: number) {
  const spinner = clack.spinner();
  spinner.start(
    `Preparing to insert ${count} records into metrics.raw_api_requests_v1 for workspace ${workspaceId} with time distribution`,
  );
  const doInsert = clickhouse.api.insert;
  let insertedCount = 0;
  let batchNumber = 0;

  try {
    while (insertedCount < count) {
      batchNumber++;
      const batchSize = Math.min(BATCH_SIZE, count - insertedCount);
      spinner.message(
        `Generating batch ${batchNumber} with realistic time distribution (${batchSize} records)...`,
      );
      const batchOfRecords = [];

      for (let i = 0; i < batchSize; i++) {
        const requestData = generateRandomApiRequest(workspaceId);
        batchOfRecords.push(requestData);
      }

      if (batchOfRecords.length > 0) {
        spinner.message(
          `Inserting batch ${batchNumber} (${insertedCount + 1}-${
            insertedCount + batchSize
          }/${count})...`,
        );

        await doInsert(batchOfRecords);
      }

      insertedCount += batchSize;
      if (batchNumber % 5 === 0 || insertedCount === count) {
        spinner.message(`Processed ${insertedCount}/${count} records...`);
      }
    }

    spinner.stop(`Successfully inserted ${count} records with realistic 30-day time distribution.`);
  } catch (error: any) {
    spinner.stop(`Error inserting data during batch ${batchNumber}: ${error.message}`);
    console.error("ClickHouse Insert Error Details:", error);
    throw error;
  }
}
