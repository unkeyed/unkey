import { readFileSync } from "node:fs";
import { createRequire } from "node:module";

const dashboard = createRequire(new URL("../../../../web/apps/dashboard/package.json", import.meta.url));
const adapter = createRequire(dashboard.resolve("@flags-sdk/vercel"));
const { evaluate } = adapter("@vercel/flags-core");

const directory = new URL("./", import.meta.url);
const data = JSON.parse(readFileSync(new URL("direct-targets.json", directory), "utf8"));
const cases = [
  ["direct", { team: { id: "matching" } }],
  ["direct", { org: { id: "matching" } }],
  ["direct", { account: { slug: "blue" } }],
  ["direct", { account: { id: "first", slug: "blue" } }],
  ["paused", {}],
];
const expected = cases.map(([flag, entities]) => {
  const result = evaluate({ definition: data.definitions[flag], environment: data.environment, entities, segments: data.segments });
  return { flag, entities, value: result.value, reason: result.reason };
});
process.stdout.write(`${JSON.stringify(expected, null, 2)}\n`);
