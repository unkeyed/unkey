import { logger, task, wait } from "@trigger.dev/sdk/v3";
import { newId } from "@unkey/id";
import { eq } from "drizzle-orm";
import { connectDatabase, schema } from "@/lib/db";

type Payload = {
  gateway: {
    id: string;
    branch: {
      id: string
    }
  };
};

export const deployTask = task({
  id: "deploy",
  run: async (payload: Payload, { ctx }) => {
    logger.log("Hello, world!", { payload, ctx });

    const db = connectDatabase()

    await wait.for({ seconds: 5 });

    return {
      message: "Hello, world!",
    };
  },
});



async function deployGateway(){
  
// import Cloudflare from "cloudflare"

async function main() {
  // const cf =new Cloudflare({
  //   token: ""
  // })

  const [_bunBin, _curentFile, gatewayBranchId] = Bun.argv;

  const buildStart = new Date();

  if (!gatewayBranchId) {
    console.log("Missing gatewayBranchId id");
    console.log("Usage: bun run main.ts gw_123");
    process.exit(1);
  }

  const branch = await db.query.gatewayBranches.findFirst({
    where: eq(schema.gatewayBranches.publicId, gatewayBranchId),
    with: {
      gateway: true,
    },
  });
  console.log("branch", branch);

  if (!branch) {
    console.log("branches", JSON.stringify(await db.query.gatewayBranches.findMany()));
    throw new Error("branch not found");
  }
  console.log(JSON.stringify(branch, null, 2));

  const deploymentId = newId("deployment");
  await db.insert(schema.gatewayDeployments).values({
    publicId: deploymentId,
    buildStart,
    gatewayId: branch.gatewayId,
    workspaceId: branch.workspaceId,
  });

  const activeDeployment = await db.query.gatewayDeployments.findFirst({
    where: eq(schema.gatewayDeployments.publicId, deploymentId),
  });
  console.log("active deployment", activeDeployment);

  if (!activeDeployment) {
    throw new Error("active deployment not found");
  }

  const wranglerConfig: {
    name: string;
    main: string;
    compatibility_date: string;
    route: { pattern: string; custom_domain: boolean };
  } = {
    name: `cus_gw__${branch.gateway.id}_${branch.id}`.replaceAll("_", "-"),
    main: "src/index.ts",
    compatibility_date: "2024-01-17",
    route: { pattern: branch.domain, custom_domain: true },
  };

  console.log("config", JSON.stringify(wranglerConfig));

  await db
    .update(schema.gatewayDeployments)
    .set({
      wranglerConfig,
    })
    .where(eq(schema.gatewayDeployments.publicId, deploymentId));
  await Bun.write(
    "../gateway/src/embed/gateway.ts",
    `
  export default {
      id: "${branch.gateway.publicId}",
      branch: {
        id: "${branch.publicId}",
        name: "${branch.name}",
      },
      name: "${branch.gateway.name}",
      origin: "${branch.origin}",
    }`,
  );

  await Bun.write("../gateway/wrangler.json", JSON.stringify(wranglerConfig, null, 2));

  console.log("running wrangler");
  const deploy = await $`pnpm wrangler deploy -j`.cwd("../gateway");
  if (deploy.stderr) {
    console.error(deploy.stderr.toString());
  }
  console.log(deploy.stdout.toString());

  const buildEnd = new Date();
  const deploymentIdRegexp = new RegExp(
    /Current Deployment ID: ([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})\n/,
  );
  const regexResult = deploymentIdRegexp.exec(deploy.stdout.toString());
  const cloudflareDeploymentId = regexResult?.at(1);

  await db
    .update(schema.gatewayDeployments)
    .set({
      buildEnd,
      cloudflareDeploymentId,
    })
    .where(eq(schema.gatewayDeployments.publicId, deploymentId));

  await db
    .update(schema.gatewayBranches)
    .set({
      activeDeploymentId: activeDeployment.id,
    })
    .where(eq(schema.gatewayBranches.id, branch.id));

  // const hostname = await cf.zoneCustomHostNames.add("6d4f6b5486c58f1211189367d1712db1", {
  //     hostname: "gw.chronark.com",
  //     ssl:{
  //       method: "txt",
  //       type:"dv",
  //       settings: {
  //         http2: "on",
  //         http3: "on",
  //         min_tls_version: "1.2",
  //         tls_1_3: "on",
  //         ciphers: [
  //           "ECDHE-RSA-AES128-GCM-SHA256",
  //           "AES128-SHA"
  //         ],
  //       },
  //       bundle_method: "ubiquitous",
  //       wildcard: false,
  //       custom_certificate: "",
  //       custom_key:""

  //       }
  // })
  // console.log(JSON.stringify({hostname},null,2))

  console.log("done", deploy.exitCode, deploymentId);
}

main();

}