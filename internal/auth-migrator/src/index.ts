import { importUsers } from "./user";
import { migrateOrganizations } from "./organization";

async function main() {
  const users = await importUsers();
  // Check if users array is not empty
  // TO be honest I'd rather run these one at a time anyways
  if (users.length > 0) {
    await migrateOrganizations();
  }
}

main().then(() => process.exit(0));
