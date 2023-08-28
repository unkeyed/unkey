import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@/lib/db";
import { redirect } from "next/navigation";

type Props = {
  params: {
    workspaceSlug: string;
  };
};

export default async function TenantOverviewPage(props: Props) {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!workspace) {
    return redirect("/onboarding");
  }

  return redirect(`/${props.params.workspaceSlug}/apis`);
}
