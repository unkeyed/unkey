import fs from "node:fs";
import * as clack from "@clack/prompts";
import { execa } from "execa";
import * as dotenv from "dotenv";

import { mysqlDrizzle, schema, eq } from "@unkey/db";
import mysql from "mysql2/promise";
import { faker } from "@faker-js/faker";
import { task } from "./util";
import { newId } from "@unkey/id";

dotenv.config();

async function main() {
	clack.intro("unkey deploy");

	const conn = await mysql.createConnection("mysql://unkey:password@localhost:3306/unkey");

	const db = mysqlDrizzle(conn, { schema, mode: "default" });
	const workspaceId = process.env.WORKSPACE_ID!;

	await db
		.insert(schema.workspaces)
		.values({
			id: workspaceId,
			name: "demo",
			features: {},
			betaFeatures: {},
			tenantId: "user_123",
		})
		.onDuplicateKeyUpdate({
			set: {
				createdAt: new Date(),
			},
		});
	const gatewayId = "gw_demo";
	await db
		.insert(schema.gateways)
		.values({
			id: gatewayId,
			name: "demo",
			workspaceId,
		})
		.onDuplicateKeyUpdate({ set: { id: gatewayId } });

	const mainBranchid = "b_main";
	await db
		.insert(schema.gatewayBranches)
		.values({
			id: mainBranchid,
			gatewayId,
			name: "main",
			workspaceId,
			domain: "demo.unkey.app",
		})
		.onDuplicateKeyUpdate({ set: { workspaceId } });

	const gitBranchName =
		process.env.FAKE_GIT_BRANCH_NAME ??
		(await execa("git", ["rev-parse", "--abbrev-ref", "HEAD"]).then(({ stdout }) => stdout));

	const gitHash = await execa("git", ["rev-parse", "--short"])
		.then(({ stdout }) => stdout)
		.catch(() => "13ff2bd8");

	await db
		.insert(schema.gatewayBranches)
		.values({
			id: newId("branch"),
			gatewayId,
			name: gitBranchName,
			workspaceId,
			domain: `${gitBranchName}-${gitHash}`.toLowerCase(),
			parentId: gitBranchName === "main" ? undefined : mainBranchid,
		})
		.onDuplicateKeyUpdate({ set: { workspaceId } });

	const branch = await db.query.gatewayBranches.findFirst({
		where: (table, { eq, and }) =>
			and(
				eq(table.workspaceId, workspaceId),
				eq(table.gatewayId, gatewayId),
				eq(table.name, gitBranchName),
			),
		with: {
			gateway: true,
		},
	});

	if (!branch) {
		console.log("branches", JSON.stringify(await db.query.gatewayBranches.findMany()));
		throw new Error("branch not found");
	}
	fs.copyFileSync("../userland/index.ts", "../user-worker/bundle/hono.ts");
	fs.writeFileSync(
		"../user-worker/bundle/secrets.ts",
		`export default {
			UNKEY_ROOT_KEY: "${process.env.UNKEY_ROOT_KEY}",
		}`,
	);

	const deploymentId = newId("deployment");
	await db.insert(schema.gatewayDeployments).values({
		id: deploymentId,
		buildStart: new Date(),
		gatewayId: branch.gatewayId,
		workspaceId: branch.workspaceId,
	});

	const activeDeployment = await db.query.gatewayDeployments.findFirst({
		where: (table, { eq }) => eq(table.id, deploymentId),
	});

	if (!activeDeployment) {
		throw new Error("active deployment not found");
	}

	const wranglerConfig = {
		name: branch.domain,
		main: "src/index.ts",
		compatibility_date: new Date().toISOString().split("T")[0],
		observability: { enabled: true },
		//	route: { pattern: branch.domain, custom_domain: true },
	};

	fs.writeFileSync("../user-worker/wrangler.json", JSON.stringify(wranglerConfig, null, 2));

	await db
		.update(schema.gatewayDeployments)
		.set({
			wranglerConfig,
		})
		.where(eq(schema.gatewayDeployments.id, deploymentId));
	const buildEnd = new Date();
	const deploymentIdRegexp = new RegExp(
		/Current Deployment ID: ([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})\n/,
	);
	await task(`Deploying ${gitBranchName} branch`, async (s) => {
		const deploy = await execa(
			"pnpm",
			["wrangler", "deploy", "--dispatch-namespace", "gateway_demo", "--name", branch.domain, "-j"],
			{
				cwd: "../user-worker",
			},
		);
		const regexResult = deploymentIdRegexp.exec(deploy.stdout.toString());
		const cloudflareDeploymentId = regexResult?.at(1);

		fs.copyFileSync(".secrets.json", "../user-worker/.secrets.json");
		await execa("pnpm", ["wrangler", "-j", "secret", "bulk", ".secrets.json"], {
			cwd: "../user-worker",
		});
		fs.unlinkSync("../user-worker/.secrets.json");

		await db
			.update(schema.gatewayDeployments)
			.set({
				buildEnd,
				cloudflareDeploymentId,
			})
			.where(eq(schema.gatewayDeployments.id, deploymentId));

		const openapiSchema = await fetch(`https://${branch.domain}.unkey.app/openapi.json`).then(
			(res) => res.json(),
		);

		await db
			.update(schema.gatewayBranches)
			.set({
				activeDeploymentId: activeDeployment.id,
				openapi: JSON.stringify(openapiSchema),
			})
			.where(eq(schema.gatewayBranches.id, branch.id));

		s.stop("done");
	});
	clack.outro(`Visit https://${branch.domain}.unkey.app`);
}

main().then(() => process.exit(0));
