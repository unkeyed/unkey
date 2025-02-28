import { Navbar } from "@/components/navbar";
import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Gauge } from "@unkey/icons";
import { redirect } from "next/navigation";
import { CreateNamespaceButton } from "./_components/create-namespace-button";
import { RatelimitClient } from "./_components/ratelimit-client";

export const dynamic = "force-dynamic";

export default async function RatelimitOverviewPage() {
  const orgId = await getOrgId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    with: {
      ratelimitNamespaces: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });

  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gauge />}>
          <Navbar.Breadcrumbs.Link href="/ratelimits">Ratelimits</Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <CreateNamespaceButton />
        </Navbar.Actions>
      </Navbar>
      <RatelimitClient ratelimitNamespaces={workspace.ratelimitNamespaces} />
    </div>
  );
}
