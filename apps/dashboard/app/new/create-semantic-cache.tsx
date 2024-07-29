"use client";

import { Button } from "@/components/ui/button";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { parseTrpcError } from "@/lib/utils";
import { PostHogEvent } from "@/providers/PostHogProvider";
import { useRouter } from "next/navigation";
import { generateSemanticCacheDefaultName } from "../(app)/semantic-cache/new/util/generate-semantic-cache-default-name";

export const CreateSemanticCacheButton: React.FC = () => {
  const router = useRouter();

  const createGateway = trpc.llmGateway.create.useMutation({
    onSuccess(res) {
      toast.success("Gateway Created", {
        description: "Your Gateway has been created",
        duration: 10_000,
      });
      PostHogEvent({
        name: "semantic_cache_gateway_created",
        properties: { id: res.id },
      });
      router.push(`/semantic-cache/${res.id}/settings`);
    },
    onError(err) {
      console.error(err);
      const message = parseTrpcError(err);
      toast.error(message);
    },
  });
  async function onClick() {
    createGateway.mutate({
      subdomain: generateSemanticCacheDefaultName(),
    });
  }
  return (
    <Button variant="primary" type="button" className="w-full" onClick={onClick}>
      Create LLM Cache Gateway
    </Button>
  );
};
