import fs from "node:fs";
import * as clack from "@clack/prompts";
import { execa } from "execa";

import { mysqlDrizzle, schema, eq } from "@unkey/db";
import mysql from "mysql2/promise";
import { faker } from "@faker-js/faker";
import { task } from "./util";
import { newId } from "@unkey/id";
async function main() {
	clack.intro("What would you like to deploy today?");

	const conn = await mysql.createConnection("mysql://unkey:password@localhost:3306/unkey");

	const db = mysqlDrizzle(conn, { schema, mode: "default" });
	const workspaceId = "ws_demo";

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

	let branches = await db.query.gatewayBranches.findMany();
	if (branches.length === 0) {
		await db.insert(schema.gatewayBranches).values({
			id: newId("branch"),
			gatewayId,
			name: "main",
			workspaceId,
			domain: "demo.unkey.app",
		});
	}

	branches = await db.query.gatewayBranches.findMany();
	let gatewayBranchId = await clack.select({
		message: "Which branch would you like to deploy to?",
		options: [
			...branches.map((b) => ({
				label: b.name,
				hint: b.activeDeploymentId ? "production" : undefined,
				value: b.id,
			})),
			{
				label: "-> new branch",
				value: null,
			},
		],
	});

	const subdomain = `${faker.hacker.adjective()}-${faker.hacker.adjective()}-${faker.science.chemicalElement().name
		}-${faker.number.int({ min: 1000, max: 9999 })}`
		.replaceAll(/\s+/g, "-")
		.toLowerCase();

	fs.copyFileSync("../userland/index.ts", "../user-worker/bundle/hono.ts");

	if (gatewayBranchId === null) {
		const name = await clack.text({ message: "What is your branch name?" });

		gatewayBranchId = newId("branch");
		await db.insert(schema.gatewayBranches).values({
			id: gatewayBranchId,
			gatewayId,
			name: name.toString(),
			workspaceId,
			domain: `${name.toString()}-demo.unkey.app`,
		});
	}
	console.log({ gatewayBranchId });
	const branch = await db.query.gatewayBranches.findFirst({
		where: (table, { eq }) => eq(table.id, gatewayBranchId as string),
		with: { gateway: true },
	});
	console.log("branch", branch);

	if (!branch) {
		console.log("branches", JSON.stringify(await db.query.gatewayBranches.findMany()));
		throw new Error("branch not found");
	}
	console.log(JSON.stringify(branch, null, 2));

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
		compatibility_date: new Date().toISOString().split("T")[0],
		route: { pattern: branch.domain, custom_domain: true },
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

	await task("Deploying", async (s) => {
		const deploy = await execa(
			"pnpm",
			["wrangler", "deploy", "--dispatch-namespace", "gateway_demo", "--name", subdomain, "-j"],
			{
				cwd: "../user-worker",
			},
		);
		const regexResult = deploymentIdRegexp.exec(deploy.stdout.toString());
		const cloudflareDeploymentId = regexResult?.at(1);

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

		s.stop("done");
	});

	clack.outro(`Visit https://${subdomain}.unkey.app`);
}

main();
