import { NamespaceNavbar } from "../namespace-navbar";
import { getWorkspaceDetails } from "../namespace.actions";
import { LogsClient } from "./components/logs-client";

export default async function RatelimitLogsPage({
  params: { namespaceId },
}: {
  params: { namespaceId: string };
}) {
  const { namespace, ratelimitNamespaces } = await getWorkspaceDetails(namespaceId);

  return (
    <div>
      <NamespaceNavbar
        activePage={{
          href: `/ratelimits/${namespace.id}/logs`,
          text: "Logs",
        }}
        namespace={namespace}
        ratelimitNamespaces={ratelimitNamespaces}
      />
      <LogsClient namespaceId={namespaceId} />
    </div>
  );
}
