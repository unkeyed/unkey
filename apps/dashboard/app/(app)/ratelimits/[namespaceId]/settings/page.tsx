import { NamespaceNavbar } from "../namespace-navbar";
import { getWorkspaceDetails } from "../namespace.actions";
import { SettingsClient } from "./components/settings-client";

export const dynamic = "force-dynamic";

type Props = {
  params: {
    namespaceId: string;
  };
};

export default async function SettingsPage(props: Props) {
  const { namespace, ratelimitNamespaces } = await getWorkspaceDetails(
    props.params.namespaceId
  );

  return (
    <div>
      <NamespaceNavbar
        activePage={{
          href: `/ratelimits/${namespace.id}/settings`,
          text: "Settings",
        }}
        namespace={namespace}
        ratelimitNamespaces={ratelimitNamespaces}
      />
      <SettingsClient namespace={namespace} />
    </div>
  );
}
