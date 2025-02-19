"use server";

import { Navbar } from "@/components/navbar";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Layers3 } from "lucide-react";
import { notFound } from "next/navigation";
import { LogsClient } from "./components/logs-client";

export default async function Page() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });

  if (!workspace) {
    return notFound();
  }

  return <LogsContainerPage />;
}

const LogsContainerPage = () => {
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
};
