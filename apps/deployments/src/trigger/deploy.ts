import fs from "node:fs";
import path from "node:path";
import { connectDatabase, schema } from "@/lib/db";
import { logger, task, wait } from "@trigger.dev/sdk/v3";
import { newId } from "@unkey/id";
import { eq } from "drizzle-orm";
import { extract } from "tar";
import { z } from "zod";

const payloadSchema = z.object({
  branchId: z.string(),
});

export const deployTask = task({
  id: "deploy",
  run: async (payload: z.infer<typeof payloadSchema>, { ctx }) => {
    const { $ } = await import("execa");

    const { branchId } = payloadSchema.parse(payload);

    logger.info("payload", { branchId });

    await $`mkdir unkey`;

    extract({
      gzip: true,
      file: "./unkey.tgz",
      sync: true,
      C: "./unkey",
    });

    const db = connectDatabase();

    const buildStart = new Date();

    const branch = await db.query.gatewayBranches.findFirst({
      where: eq(schema.gatewayBranches.id, branchId),
      with: {
        gateway: true,
      },
    });
    logger.info("branch", branch);

    if (!branch) {
      const branches = await db.query.gatewayBranches.findMany();
      logger.info("branches", { branches });
      throw new Error("branch not found");
    }
    logger.info(JSON.stringify(branch, null, 2));

    const deploymentId = newId("deployment");
    await db.insert(schema.gatewayDeployments).values({
      id: deploymentId,
      buildStart,
      gatewayId: branch.gatewayId,
      workspaceId: branch.workspaceId,
    });

    const activeDeployment = await db.query.gatewayDeployments.findFirst({
      where: eq(schema.gatewayDeployments.id, deploymentId),
    });
    logger.info("active deployment", activeDeployment);

    if (!activeDeployment) {
      throw new Error("active deployment not found");
    }

    const wranglerConfig: {
      compatibility_flags?: Array<string>;
      name: string;
      main: string;
      compatibility_date: string;
      route: { pattern: string; custom_domain: boolean };
    } = {
      compatibility_flags: ["nodejs_compat"],
      name: `customer_gateway::${branch.domain}`,
      main: "src/index.ts",
      compatibility_date: "2024-01-17",
      route: { pattern: branch.domain, custom_domain: true },
    };

    logger.info("config", { wranglerConfig });

    const cwd = "/app/unkey/apps/gateway";
    logger.info("cwd", { cwd });

    const { stdout: ls } = await $`ls`;
    logger.info(ls);

    await db
      .update(schema.gatewayDeployments)
      .set({
        wranglerConfig,
      })
      .where(eq(schema.gatewayDeployments.id, deploymentId));
    fs.writeFileSync(
      path.join(cwd, "./src/embed/gateway.ts"),
      `
  export default {
      id: "${branch.gateway.id}",
      branch: {
        id: "${branch.id}",
        name: "${branch.name}",
      },
      name: "${branch.gateway.name}",
      origin: "${branch.origin}",
    }`,
    );

    fs.writeFileSync(path.join(cwd, "./wrangler.json"), JSON.stringify(wranglerConfig, null, 2));

    logger.info("running wrangler");
    const deploy = await $({
      cwd,
    })`npx wrangler deploy -j --dispatch-namespace=staging`;
    if (deploy.stderr) {
      logger.error(deploy.stderr.toString());
    }
    if (deploy.stdout) {
      logger.info(deploy.stdout);
    }

    const buildEnd = new Date();
    const deploymentIdRegexp = new RegExp(
      /Current (Deployment|Version) ID: ([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})\n/,
    );
    const regexResult = deploymentIdRegexp.exec(deploy.stdout.toString());
    const cloudflareDeploymentId = regexResult?.at(2);

    logger.info("versionId", { cloudflareDeploymentId });

    await db
      .update(schema.gatewayDeployments)
      .set({
        buildEnd,
        cloudflareDeploymentId,
      })
      .where(eq(schema.gatewayDeployments.id, deploymentId));

    await db
      .update(schema.gatewayBranches)
      .set({
        activeDeploymentId: activeDeployment.id,
      })
      .where(eq(schema.gatewayBranches.id, branch.id));

    logger.info("done", {
      code: deploy.exitCode,
      deploymentId,
    });

    return {
      message: "Hello, world!",
    };
  },
});

// total 292
// drwxr-xr-x 1 node node 4096 May 1 09:14 .
// dr-xr-xr-x 1 root root 4096 May 1 09:14 ..
// -rw-r--r-- 1 node node 1179 May 1 09:13 Containerfile
// -rw-r--r-- 1 node node 44716 May 1 09:13 index.js
// drwxr-xr-x 122 node node 4096 May 1 09:14 node_modules
// -rw-r--r-- 1 node node 101348 May 1 09:13 package-lock.json
// -rw-r--r-- 1 node node 369 May 1 09:13 package.json
// -rw-r--r-- 1 node node 51157 May 1 09:13 worker.js
// -rw-r--r-- 1 node node 76535 May 1 09:13 worker.js.map

// total 60
// dr-xr-xr-x 1 root root 4096 May 1 09:18 .
// dr-xr-xr-x 1 root root 4096 May 1 09:18 ..
// drwxr-xr-x 1 node node 4096 May 1 09:17 app
// lrwxrwxrwx 1 root root 7 Apr 8 00:00 bin -> usr/bin
// drwxr-xr-x 2 root root 4096 Jan 28 21:20 boot
// drwxr-xr-x 5 root root 360 May 1 09:18 dev
// drwxr-xr-x 1 root root 4096 May 1 09:18 etc
// drwxr-xr-x 1 root root 4096 Apr 10 12:39 home
// lrwxrwxrwx 1 root root 7 Apr 8 00:00 lib -> usr/lib
// lrwxrwxrwx 1 root root 9 Apr 8 00:00 lib64 -> usr/lib64
// drwxr-xr-x 2 root root 4096 Apr 8 00:00 media
// drwxr-xr-x 2 root root 4096 Apr 8 00:00 mnt
// drwxr-xr-x 1 root root 4096 Apr 11 12:22 opt
// dr-xr-xr-x 845 root root 0 May 1 09:18 proc
// drwx------ 1 root root 4096 Apr 11 12:21 root
// drwxr-xr-x 1 root root 4096 May 1 09:18 run
// lrwxrwxrwx 1 root root 8 Apr 8 00:00 sbin -> usr/sbin
// drwxr-xr-x 2 root root 4096 Apr 8 00:00 srv
// dr-xr-xr-x 13 root root 0 May 1 09:18 sys
// drwxrwxrwt 1 root root 4096 Apr 11 12:22 tmp
// drwxr-xr-x 1 root root 4096 Apr 8 00:00 usr
// drwxr-xr-x 1 root root 4096 Apr 8 00:00 var
