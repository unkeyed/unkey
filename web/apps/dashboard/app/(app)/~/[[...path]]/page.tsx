// Resolves workspace-independent dashboard links against the active session workspace.
// For example, /~/apis redirects to /acme/apis, and /~/settings?tab=billing redirects to
// /acme/settings?tab=billing. This lets documentation link to dashboard pages without knowing
// each reader's workspace slug. The redirect preserves all remaining path and query parameters.
import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";

type SearchParams = Record<string, string | string[] | undefined>;

export default async function CurrentWorkspacePage({
  params,
  searchParams,
}: {
  params: Promise<{ path?: string[] }>;
  searchParams: Promise<SearchParams>;
}) {
  const { orgId } = await getAuth();
  const workspace = await db.query.workspaces.findFirst({
    columns: { slug: true },
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!workspace) {
    redirect(routes.workspaces.create());
  }

  const { path = [] } = await params;
  const pathname = [workspace.slug, ...path].join("/");
  const queryString = new URLSearchParams();

  for (const [key, value] of Object.entries(await searchParams)) {
    for (const item of Array.isArray(value) ? value : [value]) {
      if (item !== undefined) {
        queryString.append(key, item);
      }
    }
  }

  redirect(`/${pathname}${queryString.size > 0 ? `?${queryString}` : ""}`);
}
