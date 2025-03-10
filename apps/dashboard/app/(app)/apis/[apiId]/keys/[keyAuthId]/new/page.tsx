import { PageContent } from "@/components/page-content";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { CreateKey } from "./client";
import { Navigation } from "./navigation";

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
      <Navigation apiId={props.params.apiId} keyA={keyAuth} />

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
