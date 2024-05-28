import { Navbar } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import Link from "next/link";
import { redirect } from "next/navigation";

export default async function SemanticCacheLayout({ children }: { children: React.ReactNode }) {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      gateways: {
        where: (table, { isNull }) => isNull(table.deletedAt),
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });

  if (!workspace?.gateways.length) {
    return redirect("/semantic-cache/new");
  }

  const navigation = [
    {
      label: "Logs",
      href: "/semantic-cache/logs",
      segment: "logs",
    },
    {
      label: "Analytics",
      href: "/semantic-cache/analytics",
      segment: "analytics",
    },
    {
      label: "Settings",
      href: "/semantic-cache/settings",
      segment: "settings",
    },
  ];

  return (
    <div className="flex flex-col h-screen">
      <PageHeader
        title="Semantic Cache"
        description="Faster, cheaper LLM API calls through semantic caching."
      />
      <p className="text-sm text-gray-500 dark:text-gray-400">
        Your semantic cache is available at{" "}
        <span className="font-mono text-xs font-medium bg-gray-900 px-1.5 py-1 rounded-md">
          https://{workspace.gateways[0].name}.llm.unkey.dev
        </span>
        .{" "}
        <Link className="font-medium underline" href="/docs/semantic-cache">
          See how to get started
        </Link>
      </p>
      {workspace?.gateways.length && <Navbar navigation={navigation} />}
      <div className="flex-1 flex flex-col">{children}</div>
    </div>
  );
}
