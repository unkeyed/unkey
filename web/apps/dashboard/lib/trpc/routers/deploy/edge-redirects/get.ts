import {
  type WireConfig,
  type WwwMode,
  deriveWwwMode,
  domainPair,
  wireConfigSchema,
} from "@/lib/collections/deploy/edge-redirects.schema";
import { and, db, eq, inArray } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { frontlineRoutes, projects } from "@unkey/db/src/schema";
import { z } from "zod";

export type EdgeRedirectsResult = {
  wwwMode: WwwMode;
  apexDomain: string;
  wwwDomain: string;
  apexExists: boolean;
  wwwExists: boolean;
};

/**
 * getEdgeRedirects loads the joint www-handling state for the apex/www
 * pair that `domain` belongs to (the input may be either side). It reads
 * both frontline_routes rows in one query and derives the joint mode.
 *
 * Either side may be missing — common during the verification window for
 * the second domain — in which case `*Exists` flags are false and the
 * caller knows which directions are saveable.
 */
export const getEdgeRedirects = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      projectId: z.string().min(1, "Project ID is required"),
      domain: z.string().min(1, "Domain is required"),
    }),
  )
  .query(async ({ input, ctx }): Promise<EdgeRedirectsResult> => {
    const pair = domainPair(input.domain);

    const rows = await db
      .select({
        fqdn: frontlineRoutes.fullyQualifiedDomainName,
        edgeRedirectConfig: frontlineRoutes.edgeRedirectConfig,
      })
      .from(frontlineRoutes)
      .innerJoin(
        projects,
        and(eq(projects.id, frontlineRoutes.projectId), eq(projects.workspaceId, ctx.workspace.id)),
      )
      .where(
        and(
          eq(frontlineRoutes.projectId, input.projectId),
          inArray(frontlineRoutes.fullyQualifiedDomainName, [pair.apex, pair.www]),
        ),
      );

    const apexRow = rows.find((r) => r.fqdn === pair.apex);
    const wwwRow = rows.find((r) => r.fqdn === pair.www);

    return {
      wwwMode: deriveWwwMode(
        parseConfig(apexRow?.edgeRedirectConfig),
        parseConfig(wwwRow?.edgeRedirectConfig),
      ),
      apexDomain: pair.apex,
      wwwDomain: pair.www,
      apexExists: Boolean(apexRow),
      wwwExists: Boolean(wwwRow),
    };
  });

// Lenient on read: anything we don't recognize (legacy shapes, extra
// fields, malformed JSON) degrades to "no rules" with a console log.
// Rationale: the Go engine uses protojson with DiscardUnknown, so the
// dashboard should be at least as forgiving as the runtime — strict
// validation here would break the panel for any row written outside the
// current UI, including future schema versions and manual SQL writes.
// Saves still go through the strict wire schema, so we never persist
// junk; we just don't refuse to render it.
function parseConfig(raw: string | Buffer | null | undefined): WireConfig | null {
  if (!raw) {
    return null;
  }
  const text = typeof raw === "string" ? raw : Buffer.from(raw).toString();
  if (text.length === 0 || text === "{}") {
    return null;
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch (err) {
    console.warn("ignoring unparseable edge_redirect_config blob", err, text);
    return null;
  }
  const result = wireConfigSchema.safeParse(parsed);
  if (!result.success) {
    console.warn("ignoring unrecognized edge_redirect_config shape", result.error.format(), parsed);
    return null;
  }
  return result.data;
}
