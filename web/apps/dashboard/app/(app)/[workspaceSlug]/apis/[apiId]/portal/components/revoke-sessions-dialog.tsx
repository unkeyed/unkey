"use client";

import type { SessionGroup } from "@/lib/portal/sessions";
import { useRevokePortalSessions } from "@/lib/portal/use-portal-sessions";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
  toast,
} from "@unkey/ui";

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
    <AlertDialog
      open={group !== null}
      onOpenChange={(open) => {
        if (!open && revoke.isLoading) {
          return;
        }
        onOpenChange(open);
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Revoke sessions</AlertDialogTitle>
          <AlertDialogDescription>
            This ends {count === 1 ? "the 1 session" : `all ${count} sessions`}{" "}
            <span className="font-medium text-gray-12">{group?.externalId}</span> holds on this
            portal, including links that were not opened yet. They need a new link from your app to
            get back in. Their API keys keep working.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={revoke.isLoading}>Cancel</AlertDialogCancel>
          <Button
            type="button"
            variant="primary"
            color="danger"
            loading={revoke.isLoading}
            loadingLabel="Revoking sessions"
            onClick={onConfirm}
          >
            Revoke sessions
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
