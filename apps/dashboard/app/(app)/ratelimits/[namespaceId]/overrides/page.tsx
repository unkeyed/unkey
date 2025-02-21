import { PageHeader } from "@/components/dashboard/page-header";
import { Badge } from "@/components/ui/badge";
import { Empty } from "@unkey/ui";
import { CreateNewOverride } from "./create-new-override";
import { Overrides } from "./table";

export const dynamic = "force-dynamic";
export const runtime = "edge";

import { PageContent } from "@/components/page-content";
import { NamespaceNavbar } from "../namespace-navbar";
import { getWorkspaceDetailsWithOverrides } from "../namespace.actions";

export default async function OverridePage({
  params: { namespaceId },
}: {
  params: { namespaceId: string };
}) {
  const { namespace, workspace } = await getWorkspaceDetailsWithOverrides(namespaceId);

  return (
    <div>
      <NamespaceNavbar
        activePage={{
          href: `/ratelimits/${namespace.id}/overrides`,
          text: "Overrides",
        }}
        namespace={namespace}
        ratelimitNamespaces={workspace.ratelimitNamespaces}
      />
      <PageContent>
        <PageHeader
          className="m-0 mb-2"
          title="Overridden identifiers"
          actions={[
            <Badge variant="secondary" className="h-8" key="overrides">
              {Intl.NumberFormat().format(namespace.overrides?.length)} /{" "}
              {Intl.NumberFormat().format(workspace.features.ratelimitOverrides ?? 5)} used{" "}
            </Badge>,
          ]}
        />

        <CreateNewOverride namespaceId={namespace.id} />
        {namespace.overrides?.length === 0 ? (
          <Empty>
            <Empty.Icon />
            <Empty.Title>No custom ratelimits found</Empty.Title>
            <Empty.Description>Create your first override below</Empty.Description>
          </Empty>
        ) : (
          <Overrides
            workspaceId={namespace.workspaceId}
            namespaceId={namespace.id}
            ratelimits={namespace.overrides}
          />
        )}
      </PageContent>
    </div>
  );
}
