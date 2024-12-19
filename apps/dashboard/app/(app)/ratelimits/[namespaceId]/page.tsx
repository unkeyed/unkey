import { LogsClient } from "./_overview/logs-client";
import { NamespaceNavbar } from "./namespace-navbar";
import { getWorkspaceDetails } from "./namespace.actions";

export const dynamic = "force-dynamic";

export default async function RatelimitNamespacePage(props: {
  params: { namespaceId: string };
  searchParams: {
    identifier?: string;
  };
}) {
  const { namespace, ratelimitNamespaces } = await getWorkspaceDetails(
    props.params.namespaceId,
    "/ratelimits",
  );
  return (
    <div>
      <NamespaceNavbar
        activePage={{
          href: `/ratelimits/${namespace.id}`,
          text: "Requests",
        }}
        namespace={namespace}
        ratelimitNamespaces={ratelimitNamespaces}
      />
      <LogsClient namespaceId={namespace.id} />
    </div>
  );
}
