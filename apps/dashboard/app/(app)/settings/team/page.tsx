import TeamPage from "./client";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";

export const revalidate = 0;

export default async function SettingsKeysPage() {
  const tenantId = getTenantId();
  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
    with: { quota: true },
  });
  
  const team = ws?.quota?.team ?? false;

  return (
    <div>
      <TeamPage team={team}/>
    </div>
  );
}
