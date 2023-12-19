import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { UpdateTheme } from "./update-theme";
import { UpdateUserEmail } from "./update-user-email";
import { UpdateUserImage } from "./update-user-image";
import { UpdateUserName } from "./update-user-name";
export const revalidate = 0;

export default async function SettingsPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div className="mb-20 flex flex-col gap-8 ">
      <UpdateUserName />
      <UpdateUserEmail />
      <UpdateUserImage />
      <UpdateTheme />
    </div>
  );
}
