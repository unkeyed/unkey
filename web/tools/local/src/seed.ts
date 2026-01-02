import { prepareDatabase } from "./db";

async function main() {
  await prepareDatabase();
  process.exit(0);
}

main();
