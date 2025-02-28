import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { ApiListClient } from "./_components/api-list-client";
import { DEFAULT_OVERVIEW_FETCH_LIMIT } from "./_components/constants";
import { fetchApiOverview } from "./actions";
import { Navigation } from "./navigation";

export const dynamic = "force-dynamic";
export const runtime = "edge";

type Props = {
  searchParams: { new?: boolean };
};

export default async function ApisOverviewPage(props: Props) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
  });

  if (!workspace) {
    return redirect("/new");
  }

  const initialData = await fetchApiOverview({
    workspaceId: workspace.id,
    limit: DEFAULT_OVERVIEW_FETCH_LIMIT,
  });
  const unpaid = workspace.tenantId.startsWith("org_") && workspace.plan === "free";

  return (
    <div>
      <Navigation isNewApi={!!props.searchParams.new} apisLength={initialData.total} />
      <ApiListClient initialData={initialData} unpaid={unpaid} />
    </div>
  );
}
