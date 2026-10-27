"use client";

import { deliveryLabels, deliveryPending } from "@/lib/notifications/delivery";
import { useNotificationSession } from "@/lib/notifications/use-notifications";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@unkey/ui";
import { useState } from "react";

export function DestinationTest({
  destinationId,
  disabled,
  isOpen,
}: {
  destinationId: string;
  disabled: boolean;
  isOpen: boolean;
}) {
  const session = useNotificationSession();
  const [attempt, setAttempt] = useState<{ idempotencyKey: string; startedAt: number } | null>(
    null,
  );
  const query = useQuery({
    queryKey: [...session.key, "test", destinationId, attempt?.idempotencyKey],
    enabled: session.canManage && isOpen && attempt !== null,
    queryFn: async ({ signal }) => {
      if (!attempt) {
        throw new Error("A test delivery requires an idempotency key");
      }
      return (
        await getUnkeyClient().notifications.testDestination(
          { destinationId, idempotencyKey: attempt.idempotencyKey },
          { signal },
        )
      ).data;
    },
    retry: false,
    refetchOnWindowFocus: false,
    refetchInterval: (data) =>
      attempt && Date.now() - attempt.startedAt < 60_000 && data && deliveryPending(data.status)
        ? 2000
        : false,
  });
  const unfinished = attempt !== null && (!query.data || deliveryPending(query.data.status));
  return (
    <div className="flex flex-col gap-2">
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={disabled || query.isFetching}
        loading={query.isFetching}
        onClick={() =>
          unfinished
            ? query.refetch()
            : setAttempt({ idempotencyKey: crypto.randomUUID(), startedAt: Date.now() })
        }
      >
        {unfinished ? "Check test status" : attempt ? "Send another test" : "Send test"}
      </Button>
      {query.error ? (
        <p role="alert" className="text-xs text-error-11">
          {getErrorMessage(
            query.error,
            "Couldn't check the test. Check its status again without sending another message.",
          )}
        </p>
      ) : null}
      {query.data ? (
        <output className="text-xs text-gray-11">
          {deliveryLabels[query.data.status]}
          {query.data.reason ? ` ${query.data.reason}` : ""}
        </output>
      ) : null}
    </div>
  );
}
