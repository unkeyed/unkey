import { Navbar } from "@/components/navbar";
import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Layers3 } from "lucide-react";
import { notFound } from "next/navigation";
import { LogsClient } from "./components/logs-client";

export const dynamic = "force-dynamic";

export default async function Page() {
  const orgId = await getOrgId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!workspace) {
    return notFound();
  }

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Layers3 />}>
          <Navbar.Breadcrumbs.Link href="/logs">Logs</Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
      <LogsClient />
    </div>
  );
}
