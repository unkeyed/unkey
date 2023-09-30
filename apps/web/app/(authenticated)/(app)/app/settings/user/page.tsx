import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { redirect } from "next/navigation";
import { UpdateTheme } from "./update-theme";
import { UpdateUserEmail } from "./update-user-email";
import { UpdateUserImage } from "./update-user-image";
import { UpdateUserName } from "./update-user-name";
export const revalidate = 0;

export default async function SettingsPage() {
  const tenantId = getTenantId();

  const workspace = await db().query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!workspace) {
    return redirect("/onboarding");
  }

  return (
    <div className="flex flex-col gap-8 mb-20 ">
      <UpdateUserName />
      <UpdateUserEmail />
      <UpdateUserImage />
      <UpdateTheme />
    </div>
  );
}
