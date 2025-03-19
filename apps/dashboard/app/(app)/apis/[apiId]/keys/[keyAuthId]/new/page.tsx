import { PageContent } from "@/components/page-content";
import { getOrgId } from "@/lib/auth";
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
  const orgId = await getOrgId();

  const keyAuth = await db.query.keyAuth.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.keyAuthId), isNull(table.deletedAtM)),
    with: {
      workspace: true,
      api: true,
    },
  });
  if (!keyAuth || keyAuth.workspace.orgId !== orgId) {
    return notFound();
  }

  return (
    <div>
      <Navigation apiId={props.params.apiId} keyAuth={keyAuth} />

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
