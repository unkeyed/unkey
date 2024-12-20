import { CopyButton } from "@/components/dashboard/copy-button";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { getFlag } from "@/lib/utils";
import { Gauge } from "@unkey/icons";
import { Scan } from "lucide-react";
import { notFound } from "next/navigation";
import { navigation } from "../constants";
import { CreateNewOverride } from "./create-new-override";
import { Overrides } from "./table";

export const dynamic = "force-dynamic";
export const runtime = "edge";

type Props = {
  params: {
    namespaceId: string;
  };
};

export default async function OverridePage(props: Props) {
  const tenantId = getTenantId();

  const namespace = await db.query.ratelimitNamespaces.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.namespaceId), isNull(table.deletedAt)),
    with: {
      overrides: {
        columns: {
          id: true,
          identifier: true,
          limit: true,
          duration: true,
          async: true,
        },
        where: (table, { isNull }) => isNull(table.deletedAt),
      },
      workspace: {
        columns: {
          id: true,
          tenantId: true,
          features: true,
        },
      },
    },
  });
  if (!namespace || namespace.workspace.tenantId !== tenantId) {
    return notFound();
  }

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gauge />}>
          <Navbar.Breadcrumbs.Link href="/ratelimits">Ratelimits</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/ratelimits/${props.params.namespaceId}`} isIdentifier>
            {namespace.name}
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={`/ratelimits/${props.params.namespaceId}/overrides`}
            active
          >
            Overrides
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Badge
            key="namespaceId"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {namespace.id}
            <CopyButton value={namespace.id} />
          </Badge>
        </Navbar.Actions>
      </Navbar>
      <PageContent>
        <SubMenu navigation={navigation(props.params.namespaceId)} segment="overrides" />

        <div className="flex flex-col gap-8 mt-8">
          <PageHeader
            className="m-0"
            title="Overridden identifiers"
            actions={[
              <Badge variant="secondary" className="h-8">
                {Intl.NumberFormat().format(namespace.overrides.length)} /{" "}
                {Intl.NumberFormat().format(
                  getFlag(namespace.workspace, "ratelimitOverrides", {
                    devModeDefault: 5,
                  }) ?? 5,
                )}{" "}
                used{" "}
              </Badge>,
            ]}
          />

          <CreateNewOverride namespaceId={namespace.id} />
          {namespace.overrides.length === 0 ? (
            <EmptyPlaceholder>
              <EmptyPlaceholder.Icon>
                <Scan />
              </EmptyPlaceholder.Icon>
              <EmptyPlaceholder.Title>No custom ratelimits found</EmptyPlaceholder.Title>
              <EmptyPlaceholder.Description>
                Create your first override below
              </EmptyPlaceholder.Description>
            </EmptyPlaceholder>
          ) : (
            <Overrides
              workspaceId={namespace.workspace.id}
              namespaceId={namespace.id}
              ratelimits={namespace.overrides}
            />
          )}
        </div>
      </PageContent>
    </div>
  );
}
