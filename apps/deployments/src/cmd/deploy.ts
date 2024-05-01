import { connectDatabase, schema } from "@/lib/db";
import { deployTask } from "@/trigger/deploy";
import { faker } from "@faker-js/faker";
import type { GatewayBranch } from "@unkey/db";
import { newId } from "@unkey/id";
import { config } from "dotenv";
config();

async function main() {
  const db = connectDatabase();
  const gateway = await db.query.gateways.findFirst({
    where: (table, { eq }) => eq(table.id, "1"),
    with: {
      branches: {
        where: (table, { isNull }) => isNull(table.parentId),
      },
    },
  });

  if (!gateway) {
    throw new Error("gateway not found");
  }
  console.log(gateway);
  const parentBranch = gateway?.branches.at(0);
  if (!parentBranch) {
    throw new Error("parent branch not found");
  }

  const name = `${faker.word.adjective()}-${faker.animal.type()}`;
  const newBranch = {
    id: newId("branch"),
    createdAt: new Date(),
    domain: `${name}.unkey.io`,
    gatewayId: gateway.id,
    name,
    origin: parentBranch.origin,
    parentId: parentBranch.id,
    workspaceId: parentBranch.workspaceId,
    deletedAt: null,
    updatedAt: null,
    activeDeploymentId: null,
  } satisfies GatewayBranch;

  await db.insert(schema.gatewayBranches).values(newBranch);

  await deployTask.trigger({
    branchId: newBranch.id,
  });
  console.log({ newBranch });
}

main();
