import { unlinkSync } from "node:fs";
import { $ } from "bun";

const regions = [
  "us-east-1",
  "us-east-2",
  "us-west-1",
  "us-west-2",
  "af-south-1",
  "ap-east-1",
  "ap-south-2",
  "ap-southeast-3",
  "ap-southeast-4",
  "ap-south-1",
  "ap-northeast-3",
  "ap-northeast-2",
  "ap-southeast-1",
  "ap-southeast-2", // sydney
  "ca-central-1", // Canada
  "eu-central-1", // Frankfurt
  "eu-west-2", // London
  "sa-east-1", // Sao Paulo
];

const { ARTILLERY_CLOUD_API_KEY, UNKEY_KEY } = process.env;
if (!(ARTILLERY_CLOUD_API_KEY && UNKEY_KEY)) {
  console.error("missing env");
  process.exit(1);
}

async function runLambda(region: string): Promise<void> {
  const envPath = `./.env.${region}`;
  await Bun.write(
    envPath,
    `
  UNKEY_KEY="${UNKEY_KEY}"
  IDENTIFIER="${region}"
  `,
  );

  try {
    await $`artillery run-lambda --record --key=${ARTILLERY_CLOUD_API_KEY} --region=${region} --dotenv=${envPath} ./ratelimits.limit.yaml `;
  } finally {
    unlinkSync(envPath);
  }
}

async function main() {
  const ps: Promise<unknown>[] = [];
  for (const region of regions) {
    console.info("starting", region);
    const p = runLambda(region);
    ps.push(p);
    console.info("sleeping 60s");
    await new Promise((r) => setTimeout(r, 60_000));
  }
  await Promise.all(ps);
  console.info("done");
}

main();
