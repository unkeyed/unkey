import { CopyButton } from "@/components/dashboard/copy-button";
import { PageHeader } from "@/components/dashboard/page-header";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { db, schema } from "@/lib/db";
import { Gauge } from "@unkey/icons";
import { notFound } from "next/navigation";
import { UpdateCard } from "./settings";

export const dynamic = "force-dynamic";
export const runtime = "edge";

type Props = {
  params: {
    namespaceId: string;
    overrideId: string;
  };
};

export default async function OverrideSettings(props: Props) {
  const tenantId = getTenantId();

  const override = await db.query.ratelimitOverrides.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(
        eq(table.namespaceId, props.params.namespaceId),
        eq(table.id, props.params.overrideId),
        isNull(schema.keys.deletedAt),
      ),
    with: {
      workspace: true,
      namespace: true,
    },
  });
  if (!override || override.workspace.tenantId !== tenantId) {
    return notFound();
  }

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gauge />}>
          <Navbar.Breadcrumbs.Link href="/ratelimits">Ratelimits</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/ratelimits/${props.params.namespaceId}`} isIdentifier>
            {override.namespace.name}
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/ratelimits/${props.params.namespaceId}/overrides`}>
            Overrides
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={`/ratelimits/${props.params.namespaceId}/overrides/${override.id}`}
            active
            isIdentifier
          >
            {override.identifier}
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Badge
            key="namespaceId"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {override.namespace.id}
            <CopyButton value={override.namespace.id} />
          </Badge>
        </Navbar.Actions>
      </Navbar>
      <PageContent>
        <div className="flex flex-col gap-4">
          <PageHeader
            className="m-0 px-1"
            title="Identifier"
            description="Edit identifier configuration"
            actions={[
              <Badge
                key="identifier"
                variant="secondary"
                className="flex justify-between gap-2 font-mono font-medium ph-no-capture"
              >
                {override.identifier}
                <CopyButton value={override.identifier} />
              </Badge>,
            ]}
          />
          <UpdateCard
            overrideId={override.id}
            defaultValues={{
              limit: override.limit,
              duration: override.duration,
            }}
          />
        </div>
      </PageContent>
    </div>
  );
}
