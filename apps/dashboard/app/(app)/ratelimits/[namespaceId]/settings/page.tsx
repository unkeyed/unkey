import { CopyButton } from "@/components/dashboard/copy-button";
import { PageContent } from "@/components/page-content";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Code } from "@/components/ui/code";
import { NamespaceNavbar } from "../namespace-navbar";
import { getWorkspaceDetails } from "../namespace.actions";
import { DeleteNamespace } from "./delete-namespace";
import { UpdateNamespaceName } from "./update-namespace-name";

export const dynamic = "force-dynamic";

type Props = {
  params: {
    namespaceId: string;
  };
};

export default async function SettingsPage(props: Props) {
  const { namespace, ratelimitNamespaces } = await getWorkspaceDetails(props.params.namespaceId);

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
      <PageContent>
        <div className="flex flex-col gap-8">
          <UpdateNamespaceName namespace={namespace} />
          <Card>
            <CardHeader>
              <CardTitle>Namespace ID</CardTitle>
              <CardDescription>
                This is your namespace id. It's used in some API calls.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Code className="flex items-center justify-between w-full h-8 max-w-sm gap-4">
                <pre>{namespace.id}</pre>
                <div className="flex items-start justify-between gap-4">
                  <CopyButton value={namespace.id} />
                </div>
              </Code>
            </CardContent>
          </Card>
          <DeleteNamespace namespace={namespace} />
        </div>
      </PageContent>
    </div>
  );
}
