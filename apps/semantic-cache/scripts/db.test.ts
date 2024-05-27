import "dotenv/config";
import { createConnection } from "../src/db";

async function main() {
  const db = createConnection({
    host: process.env.DATABASE_HOST,
    username: process.env.DATABASE_USERNAME,
    password: process.env.DATABASE_PASSWORD,
  });

  try {
    const gateways = await db.query.gateways.findFirst();
    console.info(gateways);
  } catch (error) {
    console.error(error);
  }
}

main();
