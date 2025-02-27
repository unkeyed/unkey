"use server";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { LogsClient } from "./components/logs-client";
import { Navigation } from "./navigation";

export default async function Page() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
  });

  if (!workspace) {
    return notFound();
  }

  return (
    <div>
      <Navigation />
      <LogsClient />
    </div>
  );
}
