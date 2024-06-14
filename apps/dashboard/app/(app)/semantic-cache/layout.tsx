import { PageHeader } from "@/components/dashboard/page-header";

export default async function SemanticCacheLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col h-screen">
      <PageHeader
        title="Semantic Cache"
        description="Faster, cheaper LLM API calls through semantic caching."
      />
      <div className="flex-1 flex flex-col">{children}</div>
    </div>
  );
}
