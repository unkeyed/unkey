import { Navbar } from "@/components/dashboard/navbar";
import { PageHeader } from "@/components/dashboard/page-header";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import Link from "next/link";
import { redirect } from "next/navigation";

export default async function SemanticCacheLayout({ children }: { children: React.ReactNode }) {
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
      <Navbar navigation={navigation} />
      <div className="flex-1 flex flex-col">{children}</div>
    </div>
  );
}
