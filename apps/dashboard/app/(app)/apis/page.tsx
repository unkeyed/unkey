import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { ApiListClient } from "./_components/api-list-client";
import { DEFAULT_OVERVIEW_FETCH_LIMIT } from "./_components/constants";
import { fetchApiOverview } from "./actions";
import { Navigation } from "./navigation";

export const dynamic = "force-dynamic";

type Props = {
  searchParams: { new?: boolean };
};

export default async function ApisOverviewPage(props: Props) {
  const orgId = await getOrgId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!workspace) {
    return redirect("/new");
  }

  const initialData = await fetchApiOverview({
    workspaceId: workspace.id,
    limit: DEFAULT_OVERVIEW_FETCH_LIMIT,
  });

  return (
    <div>
      <Navigation isNewApi={!!props.searchParams.new} apisLength={initialData.total} />
      <ApiListClient initialData={initialData} />
    </div>
  );
}
