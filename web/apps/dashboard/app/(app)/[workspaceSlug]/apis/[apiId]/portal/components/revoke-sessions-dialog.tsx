"use client";

import type { SessionGroup } from "@/lib/portal/sessions";
import { useRevokePortalSessions } from "@/lib/portal/use-portal-sessions";
import { Button, DialogContainer, toast } from "@unkey/ui";

export function RevokeSessionsDialog({
  portalId,
  group,
  onOpenChange,
}: {
  portalId: string;
  group: SessionGroup | null;
  onOpenChange: (open: boolean) => void;
}) {
  const revoke = useRevokePortalSessions(portalId);

  const onConfirm = async () => {
    if (!group) {
      return;
    }
    try {
      const { sessionsRevoked } = await revoke.mutateAsync({ externalId: group.externalId });
      onOpenChange(false);
      if (sessionsRevoked === 0) {
        toast.info(`${group.externalId} had no sessions left to revoke`);
        return;
      }
      toast.success(
        `Revoked ${sessionsRevoked} ${sessionsRevoked === 1 ? "session" : "sessions"} for ${group.externalId}`,
      );
    } catch {
      // The hook surfaced the failure; keep the dialog open to retry.
    }
  };

  const count = group?.sessions.length ?? 0;

  return (
    <DialogContainer
      isOpen={group !== null}
      onOpenChange={(open) => {
        if (!open && revoke.isLoading) {
          return;
        }
        onOpenChange(open);
      }}
      title="Revoke sessions"
      subTitle="Sign this user out of the portal"
      footer={
        <Button
          type="button"
          variant="primary"
          color="danger"
          size="xlg"
          className="w-full rounded-lg"
          disabled={revoke.isLoading}
          loading={revoke.isLoading}
          loadingLabel="Revoking sessions"
          onClick={onConfirm}
        >
          Revoke sessions
        </Button>
      }
    >
      <p className="text-sm leading-6 text-gray-11">
        This ends {count === 1 ? "the 1 session" : `all ${count} sessions`}{" "}
        <span className="font-medium text-gray-12">{group?.externalId}</span> holds on this portal,
        including links that were not opened yet. They need a new link from your app to get back in.
        Their API keys keep working.
      </p>
    </DialogContainer>
  );
}
