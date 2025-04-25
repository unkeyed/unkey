import { PageContent } from "@/components/page-content";
import { getOrgId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { notFound, redirect } from "next/navigation";
import { ApisNavbar } from "../api-id-navbar";
import { SettingsClient } from "./components/settings-client";
export const dynamic = "force-dynamic";

type Props = {
  params: {
    apiId: string;
  };
};

export default async function SettingsPage(props: Props) {
  const orgId = await getOrgId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    with: {
      apis: {
        where: eq(schema.apis.id, props.params.apiId),
        with: {
          keyAuth: true,
        },
      },
    },
  });
  if (!workspace || workspace.orgId !== orgId) {
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
      <ApisNavbar
        api={api}
        activePage={{
          href: `/apis/${api.id}/settings`,
          text: "Settings",
        }}
        apis={workspace.apis}
      />
      <PageContent>
        <SettingsClient api={api} workspace={workspace} keyAuth={keyAuth} />
      </PageContent>
    </div>
  );
}
