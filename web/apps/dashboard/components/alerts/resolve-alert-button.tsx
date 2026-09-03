"use client";

import { trpc } from "@/lib/trpc/client";
import { Button, DialogContainer, FormTextarea, toast } from "@unkey/ui";
import { useState } from "react";

export function ResolveAlertButton({ alertId }: { alertId: string }) {
  const [isOpen, setIsOpen] = useState(false);
  const [message, setMessage] = useState("");
  const utils = trpc.useUtils();
  const resolveAlert = trpc.alerts.resolve.useMutation({
    onSuccess: async () => {
      await Promise.all([
        utils.alerts.list.invalidate(),
        utils.alerts.get.invalidate({ alertId }),
        utils.alerts.summary.invalidate(),
        utils.alerts.series.invalidate(),
      ]);
      setMessage("");
      setIsOpen(false);
      toast.success("Alert resolved");
    },
    onError: (error) => toast.error(error.message),
  });
  const validMessage = message.trim();

  return (
    <div
      className="relative z-20"
      onClick={(event) => event.stopPropagation()}
      onKeyDown={(event) => event.stopPropagation()}
    >
      <Button size="sm" variant="outline" onClick={() => setIsOpen(true)}>
        Resolve
      </Button>
      <DialogContainer
        isOpen={isOpen}
        onOpenChange={(open) => {
          if (!resolveAlert.isLoading) {
            setIsOpen(open);
          }
        }}
        title="Resolve anomaly alert"
        subTitle="Record what caused the anomaly and how you responded."
        footer={
          <div className="flex w-full gap-2">
            <Button
              className="flex-1"
              variant="outline"
              disabled={resolveAlert.isLoading}
              onClick={() => setIsOpen(false)}
            >
              Cancel
            </Button>
            <Button
              className="flex-1"
              variant="primary"
              disabled={!validMessage}
              loading={resolveAlert.isLoading}
              onClick={() => resolveAlert.mutate({ alertId, message: validMessage })}
            >
              Resolve alert
            </Button>
          </div>
        }
      >
        <FormTextarea
          autoFocus
          required
          requirement="required"
          label="What happened / what did you do?"
          description={`${message.length}/1000 characters`}
          value={message}
          maxLength={1000}
          rows={5}
          placeholder="Describe the cause and the action taken."
          onChange={(event) => setMessage(event.target.value)}
        />
      </DialogContainer>
    </div>
  );
}
