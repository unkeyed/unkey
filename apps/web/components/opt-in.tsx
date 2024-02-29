"use client";

import { Button } from "@/components/ui/button";
import { trpc } from "@/lib/trpc/client";
import { type Workspace } from "@unkey/db";
import { Power } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import React from "react";
import { EmptyPlaceholder } from "./dashboard/empty-placeholder";

import { toast } from "@/components/ui/toaster";
import { PostHogEvent } from "@/providers/PostHogProvider";
import { Loading } from "./dashboard/loading";
type Props = {
  title: string;
  description: string;
  feature: keyof Workspace["betaFeatures"];
};

export const OptIn: React.FC<Props> = ({ title, description, feature }) => {
  const router = useRouter();
  const optIn = trpc.workspace.optIntoBeta.useMutation({
    onMutate() {
      PostHogEvent({
        name: "self-serve-opt-in",
        properties: { feature },
      });
    },
    onSuccess() {
      PostHogEvent({
        name: "self-serve-opt-in",
        properties: { feature },
      });
      toast.success("Successfully opted in");
      router.refresh();
    },
  });
  return (
    <EmptyPlaceholder className="h-full">
      <EmptyPlaceholder.Icon>
        <Power />
      </EmptyPlaceholder.Icon>
      <EmptyPlaceholder.Title>{title}</EmptyPlaceholder.Title>
      <EmptyPlaceholder.Description>{description}</EmptyPlaceholder.Description>

      <div className="flex items-center gap-4">
        <Link href={`mailto:support@unkey.dev?subject=Beta Access: ${feature}`}>
          <Button variant="secondary">Get in touch</Button>
        </Link>

        <Button variant="primary" onClick={() => optIn.mutate({ feature })}>
          {optIn.isLoading ? <Loading /> : "Opt In"}
        </Button>
      </div>
    </EmptyPlaceholder>
  );
};
