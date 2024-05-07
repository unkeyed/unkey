"use client";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { useRouter } from "next/navigation";
import type React from "react";
import { useState } from "react";

type Props = {
  webhook: {
    id: string;
    enabled: boolean;
  };
};

export const ToggleWebhookButton: React.FC<Props> = ({ webhook }) => {
  const router = useRouter();
  const [optimisticEnabled, setOptimisticEnabled] = useState(webhook.enabled);

  const toggle = trpc.webhook.toggle.useMutation({
    onMutate(variables) {
      setOptimisticEnabled(variables.enabled);
    },
    onSuccess() {
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function action(enabled: boolean) {
    toast.promise(toggle.mutateAsync({ webhookId: webhook.id, enabled }), {
      loading: `${enabled ? "Enabling" : "Disabling"} your webhook...`,
      success: (res) => `Your webhook has been ${res.enabled ? "enabled" : "disabled"}`,
    });
  }
  return (
    <div className="flex items-center gap-2">
      <Label>{optimisticEnabled ? "Enabled" : "Disabled"}</Label>
      <Switch checked={optimisticEnabled} onCheckedChange={action} />
    </div>
  );
};
