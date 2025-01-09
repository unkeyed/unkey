import { CopyButton } from "@/components/dashboard/copy-button";
import { Empty } from "@unkey/ui";

import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { Code } from "@/components/ui/code";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Gauge } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { BookOpen } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { Suspense } from "react";
import { RatelimitCard } from "./card";
import { CreateNamespaceButton } from "./create-namespace-button";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function RatelimitOverviewPage() {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      ratelimitNamespaces: {
        where: (table, { isNull }) => isNull(table.deletedAt),
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

  const snippet = `curl -XPOST 'https://api.unkey.dev/v1/ratelimits.limit' \\
  -H 'Content-Type: application/json' \\
  -H 'Authorization: Bearer <UNKEY_ROOT_KEY>' \\
  -d '{
      "namespace": "demo_namespace",
      "identifier": "user_123",
      "limit": 10,
      "duration": 10000
  }'`;

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
      <PageContent>
        {workspace.ratelimitNamespaces.length > 0 ? (
          <ul className="grid grid-cols-1 gap-x-6 gap-y-8 lg:grid-cols-2 ">
            {workspace.ratelimitNamespaces.map((namespace) => (
              <Link key={namespace.id} href={`/ratelimits/${namespace.id}`}>
                <Suspense fallback={null}>
                  <RatelimitCard namespace={namespace} workspace={workspace} />
                </Suspense>
              </Link>
            ))}
          </ul>
        ) : (
          <Empty>
            <Empty.Icon />
            <Empty.Title>No Namespaces found</Empty.Title>
            <Empty.Description>
              You haven&apos;t created any Namespaces yet. Create one by performing a limit request
              as shown below.
            </Empty.Description>
            <Code className="flex items-start gap-8 p-4 my-8 text-xs text-left">
              {snippet}
              <CopyButton value={snippet} />
            </Code>
            <Empty.Actions>
              <Link href="/docs/ratelimiting/introduction" target="_blank">
                <Button className="items-center w-full gap-2 ">
                  <BookOpen className="w-4 h-4 " />
                  Read the docs
                </Button>
              </Link>
            </Empty.Actions>
          </Empty>
        )}
      </PageContent>
    </div>
  );
}
