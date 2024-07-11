"use client";

import { Button } from "@/components/ui/button";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { PostHogEvent, PostHogIdentify } from "@/providers/PostHogProvider";
import { useUser } from "@clerk/nextjs";
import { useRouter } from "next/navigation";
import { z } from "zod";
import { generateSemanticCacheDefaultName } from "../(app)/semantic-cache/new/util/generate-semantic-cache-default-name";

type Props = any;

export const CreateSemanticCacheButton: React.FC<Props> = () => {
  const { user, isLoaded } = useUser();
  const router = useRouter();

  if (isLoaded && user) {
    PostHogIdentify({ user });
  }
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
      toast.error("An error occured", {
        description: err.message,
      });
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
