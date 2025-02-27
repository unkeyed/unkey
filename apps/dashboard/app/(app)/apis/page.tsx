import { CreateApiButton } from "./create-api-button";

import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { getTenantId } from "@/lib/auth";
import { and, db, eq, isNull, schema, sql } from "@/lib/db";
import { Nodes } from "@unkey/icons";
import { redirect } from "next/navigation";
import { ApiList } from "./client";

export const dynamic = "force-dynamic";

type Props = {
  searchParams: { new?: boolean };
};

export default async function ApisOverviewPage(props: Props) {
  const tenantId = await getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
    with: {
      apis: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
      },
    },
  });

  if (!workspace) {
    return redirect("/new");
  }

  const apis = await Promise.all(
    workspace.apis.map(async (api) => ({
      id: api.id,
      name: api.name,
      keys: await db
        .select({ count: sql<number>`count(*)` })
        .from(schema.keys)
        .where(and(eq(schema.keys.keyAuthId, api.keyAuthId!), isNull(schema.keys.deletedAtM))),
    })),
  );

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Nodes />}>
          <Navbar.Breadcrumbs.Link href="/apis">APIs</Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <CreateApiButton
            key="createApi"
            defaultOpen={apis.length === 0 || props.searchParams.new}
          />
        </Navbar.Actions>
      </Navbar>
      <PageContent>
          <ApiList apis={apis} />
      </PageContent>
    </div>
  );
}
