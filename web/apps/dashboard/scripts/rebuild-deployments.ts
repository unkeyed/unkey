#!/usr/bin/env bun
/**
 * Admin recovery: force-rebuild deployments whose Docker images are
 * unavailable (e.g. accidentally deleted from the registry).
 *
 * For each deployment ID, calls OpsService.RebuildDeployment on the prod
 * control plane. The server loads the source's project/app/env/git SHA,
 * enforces guardrails (git connection required, no newer sibling unless
 * forced), and kicks off a fresh build.
 *
 * Dry-run by default — prints the IDs that would be rebuilt. Pass
 * --execute to actually call ctrl.
 *
 * Batches with a configurable pause between calls so we don't stampede
 * the build slot pool.
 *
 * Usage:
 *   CTRL_BEARER=... bun run scripts/rebuild-deployments.ts \
 *     --ctrl https://control.unkey.cloud \
 *     --ids @/tmp/rebuild-ids.txt \
 *     --reason "image lost after pod move" \
 *     --batch-size 5 \
 *     --pause-ms 30000 \
 *     --execute
 */

import { readFileSync } from "node:fs";
import { parseArgs } from "node:util";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { OpsService } from "../gen/proto/ctrl/v1/ops_pb";

async function main() {
  const { values } = parseArgs({
    options: {
      ctrl: { type: "string" },
      ids: { type: "string" },
      reason: { type: "string", default: "manual rebuild" },
      "batch-size": { type: "string", default: "1" },
      "pause-ms": { type: "string", default: "0" },
      force: { type: "boolean", default: false },
      execute: { type: "boolean", default: false },
    },
    strict: true,
  });

  const ctrlUrl = values.ctrl;
  const idsArg = values.ids;
  const dryRun = !values.execute;
  const batchSize = Math.max(1, Number.parseInt(values["batch-size"], 10) || 1);
  const pauseMs = Math.max(0, Number.parseInt(values["pause-ms"], 10) || 0);

  if (!ctrlUrl || !idsArg) {
    console.error("missing required flags: --ctrl, --ids");
    process.exit(2);
  }

  const bearer = process.env.CTRL_BEARER;
  if (!dryRun && !bearer) {
    console.error("CTRL_BEARER env var is required when --execute is set");
    process.exit(2);
  }

  const ids = loadIds(idsArg);
  if (ids.length === 0) {
    console.error("no deployment ids provided");
    process.exit(2);
  }

  const client = createClient(
    OpsService,
    createConnectTransport({
      baseUrl: ctrlUrl,
      interceptors: bearer
        ? [
            (next) => (req) => {
              req.header.set("Authorization", `Bearer ${bearer}`);
              return next(req);
            },
          ]
        : [],
    }),
  );

  const mode = dryRun ? "DRY-RUN" : "EXECUTE";
  console.info(
    `mode=${mode} ctrl=${ctrlUrl} count=${ids.length} batch=${batchSize} pause=${pauseMs}ms force=${values.force}\n`,
  );

  let ok = 0;
  let failed = 0;

  for (let i = 0; i < ids.length; i += batchSize) {
    const batch = ids.slice(i, i + batchSize);
    const results = await Promise.allSettled(
      batch.map(async (id, idx) => {
        const label = `[${i + idx + 1}/${ids.length}] ${id}`;
        if (dryRun) {
          console.info(`${label} (dry-run, would rebuild with reason="${values.reason}")`);
          return { id, newId: null as string | null };
        }
        const res = await client.rebuildDeployment({
          deploymentId: id,
          reason: values.reason ?? "",
          force: values.force ?? false,
        });
        console.info(`${label} NEW deployment_id=${res.deploymentId}`);
        return { id, newId: res.deploymentId };
      }),
    );

    for (const r of results) {
      if (r.status === "fulfilled") {
        ok++;
      } else {
        failed++;
        console.info(`  FAIL: ${r.reason instanceof Error ? r.reason.message : String(r.reason)}`);
      }
    }

    if (pauseMs > 0 && i + batchSize < ids.length) {
      await new Promise((resolve) => setTimeout(resolve, pauseMs));
    }
  }

  console.info(`\ndone: ok=${ok} failed=${failed}`);
  if (failed > 0) {
    process.exit(1);
  }
}

function loadIds(arg: string): string[] {
  let raw = arg;
  if (arg.startsWith("@")) {
    raw = readFileSync(arg.slice(1), "utf8");
  }
  const seen = new Set<string>();
  const out: string[] = [];
  for (const token of raw.split(/[\s,]+/)) {
    const id = token.trim();
    if (!id || seen.has(id)) {
      continue;
    }
    seen.add(id);
    out.push(id);
  }
  return out;
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
