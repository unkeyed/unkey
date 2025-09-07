import { Navigation } from "@/components/navigation/navigation";
import { InputSearch } from "@unkey/icons";
import { getWorkspace } from "./actions";
import { LogsClient } from "./components/logs-client";
export const dynamic = "force-dynamic";

import { getAuthOrRedirect } from "@/lib/auth";
import { redirect } from "next/navigation";
export default async function AuditPage() {
  const { orgId } = await getAuthOrRedirect();
  if (!orgId) {
    redirect("/new");
  }
  const { workspace, members } = await getWorkspace(orgId);

  return (
    <div>
      <Navigation href="/audit" name="Audit" icon={<InputSearch />} />
      <LogsClient rootKeys={workspace.keys} buckets={["unkey_mutations"]} members={members} />
    </div>
  );
}
