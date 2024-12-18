import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Nodes } from "@unkey/icons";
import { notFound } from "next/navigation";
import { CreateKey } from "./client";

export default async function CreateKeypage(props: {
  params: {
    apiId: string;
    keyAuthId: string;
  };
}) {
  const tenantId = await getTenantId();

  const keyAuth = await db.query.keyAuth.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.keyAuthId), isNull(table.deletedAtM)),
    with: {
      workspace: true,
      api: true,
    },
  });
  if (!keyAuth || keyAuth.workspace.tenantId !== tenantId) {
    return notFound();
  }

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Nodes />}>
          <Navbar.Breadcrumbs.Link href="/apis">APIs</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/apis/${props.params.apiId}`} isIdentifier>
            {keyAuth.api.name}
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/apis/${props.params.apiId}/keys/${keyAuth.id}`}>
            Keys
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            active
            href={`/apis/${props.params.apiId}/keys/${keyAuth.id}/new`}
          >
            Create new key
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Badge
            key="apiId"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {keyAuth.api.id}
            <CopyButton value={keyAuth.api.id} />
          </Badge>
        </Navbar.Actions>
      </Navbar>

      <PageContent>
        <CreateKey
          keyAuthId={keyAuth.id}
          apiId={props.params.apiId}
          defaultBytes={keyAuth.defaultBytes}
          defaultPrefix={keyAuth.defaultPrefix}
        />
      </PageContent>
    </div>
  );
}
