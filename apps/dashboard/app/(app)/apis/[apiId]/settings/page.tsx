import { CopyButton } from "@/components/dashboard/copy-button";
import { CreateKeyButton } from "@/components/dashboard/create-key-button";
import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Code } from "@/components/ui/code";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { Nodes } from "@unkey/icons";
import { notFound, redirect } from "next/navigation";
import { navigation } from "../constants";
import { DefaultBytes } from "./default-bytes";
import { DefaultPrefix } from "./default-prefix";
import { DeleteApi } from "./delete-api";
import { DeleteProtection } from "./delete-protection";
import { UpdateApiName } from "./update-api-name";
import { UpdateIpWhitelist } from "./update-ip-whitelist";

export const dynamic = "force-dynamic";

type Props = {
  params: {
    apiId: string;
  };
};

export default async function SettingsPage(props: Props) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      apis: {
        where: eq(schema.apis.id, props.params.apiId),
        with: {
          keyAuth: true,
        },
      },
    },
  });
  if (!workspace || workspace.tenantId !== tenantId) {
    return redirect("/new");
  }

  const api = workspace.apis.find((api) => api.id === props.params.apiId);
  if (!api) {
    return notFound();
  }

  const keyAuth = api.keyAuth;
  if (!keyAuth) {
    return notFound();
  }

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Nodes />}>
          <Navbar.Breadcrumbs.Link href="/apis">APIs</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/apis/${props.params.apiId}`} isIdentifier>
            {api.name}
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link active href={`/apis/${props.params.apiId}/settings`}>
            Settings
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Badge
            key="apiId"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {api.id}
            <CopyButton value={api.id} />
          </Badge>
          <CreateKeyButton apiId={api.id} keyAuthId={api.keyAuthId!} />
        </Navbar.Actions>
      </Navbar>

      <PageContent>
        <SubMenu navigation={navigation(api.id, api.keyAuthId!)} segment="settings" />

        <div className="flex flex-col gap-8 mb-20 mt-8">
          <UpdateApiName api={api} />
          <DefaultBytes keyAuth={keyAuth} />
          <DefaultPrefix keyAuth={keyAuth} />
          <UpdateIpWhitelist api={api} workspace={workspace} />
          <Card>
            <CardHeader>
              <CardTitle>API ID</CardTitle>
              <CardDescription>This is your api id. It's used in some API calls.</CardDescription>
            </CardHeader>
            <CardContent>
              <Code className="flex items-center justify-between w-full h-8 max-w-sm gap-4">
                <pre>{api.id}</pre>
                <div className="flex items-start justify-between gap-4">
                  <CopyButton value={api.id} />
                </div>
              </Code>
            </CardContent>
          </Card>
          <DeleteProtection api={api} />
          <DeleteApi api={api} keys={keyAuth.sizeApprox} />
        </div>
      </PageContent>
    </div>
  );
}
