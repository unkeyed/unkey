import { $ } from "bun";

const regions = [
  "ams",
  "iad",
  "atl",
  "bog",
  "bos",
  "otp",
  "ord",
  "dfw",
  "den",
  "eze",
  // "fra",
  "gdl",
  "hkg",
  "jnb",
  "lhr",
  "lax",
  "mad",
  "mia",
  "yul",
  "bom",
  "cdg",
  "phx",
  "qro",
  "gig",
  "sjc",
  "scl",
  "gru",
  "sea",
  "ewr",
  "sin",
  "arn",
  "syd",
  "nrt",
  "yyz",
  "waw",
];

const { ARTILLERY_CLOUD_API_KEY, UNKEY_KEY, FLY_API_KEY } = process.env;
if (!ARTILLERY_CLOUD_API_KEY || !UNKEY_KEY || !FLY_API_KEY) {
  console.error("missing env");
  process.exit(1);
}

const FLY_APP_NAME = "artillery";
const ARTILLERY_YAML_FILE = "./keys.verifyKey.yaml";

async function runMachine(image: string, region: string): Promise<void> {
  await fetch(`https://api.machines.dev/v1/apps/${FLY_APP_NAME}/machines`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${FLY_API_KEY}`,
    },
    body: JSON.stringify({
      region,
      config: {
        init: {
          exec: [
            "artillery",
            "run",
            ARTILLERY_YAML_FILE,
            "--record",
            "--target=https://canary.unkey.dev",
            `--tags=platform:fly,region:${region}`,
          ],
        },
        image,
        restart: {
          policy: "no",
        },
        env: {
          ARTILLERY_CLOUD_API_KEY,
          UNKEY_KEY,
        },
        auto_destroy: true,
        size: "performance-2x",
      },
    }),
  }).then(async (res) => {
    console.log(`Machine ${(await res.json()).id} started in ${region}`);
  });
}

async function main() {
  const build = await $`fly deploy --build-only --push --access-token=${FLY_API_KEY}`;

  const imageRes = new RegExp("naming to (.+) done\n").exec(build.stdout.toString());
  if (!imageRes) {
    throw new Error("image not detected");
  }

  const image = imageRes[1];

  await Promise.all(
    regions.map(async (region) => {
      await runMachine(image, region);
    }),
  );
  console.log("done");
}

main();
