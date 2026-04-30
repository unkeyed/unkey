import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";
import { type axiomConfigSchema, drainConfigSchema } from "./schemas";

// testPush runs a single synthetic record through the customer-supplied
// provider config + token. It is the dashboard wizard's "test connection"
// button: any auth, dataset, or endpoint error surfaces verbatim before
// the drain row is ever persisted, so customers fix the obvious
// misconfigurations at setup time instead of staring at a paused drain
// later.
//
// We do NOT route this through the logdrain Go service. Reasons:
//   1. The test push has no row to associate with — at create-wizard
//      time the drain does not exist yet.
//   2. Round-tripping through the coordinator would mean either a new
//      synchronous RPC surface on logdrain (extra wire), or persisting a
//      throwaway drain row first (extra cleanup). Neither is worth it
//      for a one-shot HTTP call.
//   3. The wire format is identical to what the coordinator sends, so
//      "if testPush succeeds, the coordinator will succeed" holds.

const inputSchema = z
  .object({
    credential: z.string().trim().min(1),
  })
  .and(drainConfigSchema);

type Result = { ok: true } | { ok: false; status?: number; error: string };

export const testPushLogDrain = workspaceProcedure
  .input(inputSchema)
  .mutation(async ({ ctx, input }): Promise<Result> => {
    try {
      switch (input.provider) {
        case "axiom":
          return await pushAxiom(input.config, input.credential, ctx.workspace.id as string);
      }
    } catch (error) {
      // Network or unexpected errors fall through as "error" rather than
      // 500 — the dashboard treats this as a soft failure and lets the
      // user retry without a page reload.
      return {
        ok: false,
        error: error instanceof Error ? error.message : "test push failed",
      };
    }
    // Unreachable, but TypeScript narrowing on the discriminated union
    // does not see the exhaustive return.
    throw new TRPCError({
      code: "BAD_REQUEST",
      message: "Unknown provider",
    });
  });

// Synthetic record matches the shape the coordinator sends in its
// HealthCheck path so a passing testPush is genuine evidence.
function syntheticEvent(workspaceId: string) {
  return {
    timestamp: Date.now(),
    workspace_id: workspaceId,
    project_id: "proj_healthcheck",
    source: "runtime",
    level: "info",
    message: "unkey logdrain test push",
    attributes: { "unkey.healthcheck": true },
  };
}

async function pushAxiom(
  cfg: z.infer<typeof axiomConfigSchema>,
  token: string,
  workspaceId: string,
): Promise<Result> {
  const endpoint = cfg.endpoint || "https://api.axiom.co";
  const url = `${endpoint}/v1/datasets/${encodeURIComponent(cfg.dataset)}/ingest`;
  const ev = syntheticEvent(workspaceId);
  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify([{ _time: new Date(ev.timestamp).toISOString(), ...ev }]),
  });
  if (!res.ok) {
    return { ok: false, status: res.status, error: await res.text() };
  }
  return { ok: true };
}
