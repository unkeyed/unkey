import { prepareDatabase } from "./db";

async function main() {
  await prepareDatabase(process.env.DRIZZLE_DATABASE_URL);
  process.exit(0);
}

main();
