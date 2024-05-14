import { PageHeader } from "@/components/dashboard/page-header";
import { Separator } from "@/components/ui/separator";

export default function SemanticCachePage() {
  return (
    <div>
      <PageHeader
        title="Semantic Cache"
        description="Faster, cheaper LLM API calls through semantic caching"
      />
      <Separator className="my-6" />
    </div>
  );
}
