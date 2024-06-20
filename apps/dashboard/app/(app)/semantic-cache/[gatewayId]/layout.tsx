import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";

export default async function SemanticCacheLayout({
  params,
  children,
}: { params: { gatewayId: string }; children: React.ReactNode }) {
  const navigation = [
    {
      label: "Logs",
      href: `/semantic-cache/${params.gatewayId}/logs`,
      segment: "logs",
    },
    {
      label: "Analytics",
      href: `/semantic-cache/${params.gatewayId}/analytics`,
      segment: "analytics",
    },
    {
      label: "Settings",
      href: `/semantic-cache/${params.gatewayId}/settings`,
      segment: "settings",
    },
  ];

  const tenantId = await getTenantId();
  const ws = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, tenantId),
    with: {
      llmGateways: {
        where: (table, { eq }) => eq(table.id, params.gatewayId),
      },
    },
  });
  const gateway = ws?.llmGateways.at(0);
  if (!gateway) {
    return notFound();
  }

  const gatewayUrl = `https://${gateway.name}.llm.unkey.dev`;

  return (
    <>
      <PageHeader
        title="Semantic Cache"
        description="Faster, cheaper LLM API calls through semantic caching."
        actions={[
          <Badge
            key="gateway"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {gatewayUrl}
            <CopyButton value={gatewayUrl} />
          </Badge>,
        ]}
      />
      <Navbar navigation={navigation} />
      <div className="mt-8">{children}</div>
    </>
  );
}
