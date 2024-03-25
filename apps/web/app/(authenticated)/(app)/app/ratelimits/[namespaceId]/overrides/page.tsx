import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { PageHeader } from "@/components/dashboard/page-header";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Scan } from "lucide-react";
import { notFound } from "next/navigation";
import { CreateNewOverride } from "./create-new-override";
import { Table } from "./table";

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
        },
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
    <>
      <PageHeader
        title="Overridden identifiers"
        actions={[
          <Badge variant="secondary" className="h-8">
            {Intl.NumberFormat().format(namespace.overrides.length)} /{" "}
            {Intl.NumberFormat().format(namespace.workspace.features.ratelimitOverrides ?? 5)} used{" "}
          </Badge>,
        ]}
      />
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
        <Table
          workspaceId={namespace.workspace.id}
          namespaceId={namespace.id}
          ratelimits={namespace.overrides}
        />
      )}

      <Separator className="my-8" />
      <CreateNewOverride namespaceId={namespace.id} />
    </>
  );
}
