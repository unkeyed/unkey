import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
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
      {unpaid ? (
        <div className="h-screen flex items-center justify-center">
          <div className="flex justify-center items-center">
            <Empty className="border border-gray-6 rounded-lg bg-gray-1">
              <Empty.Title className="text-xl">Upgrade your plan</Empty.Title>
              <Empty.Description>
                Team workspaces is a paid feature. Please switch to a paid plan to continue using
                it.
              </Empty.Description>
              <Empty.Actions className="mt-4 ">
                <a href="/settings/billing" target="_blank" rel="noopener noreferrer">
                  <Button>
                    <BookBookmark />
                    Subscribe
                  </Button>
                </a>
              </Empty.Actions>
            </Empty>
          </div>
        </div>
      ) : (
        <ApiListClient initialData={initialData} />
      )}
    </div>
  );
}
