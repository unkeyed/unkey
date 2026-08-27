"use client";

import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { SecretKeyDialog } from "@/components/secret-key-dialog";
import { Button, Dialog, DialogContent, DialogTitle, FormCheckbox, FormSelect } from "@unkey/ui";
import { useState } from "react";
import {
  DEFAULT_GRACE_PERIOD,
  GRACE_PERIOD_OPTIONS,
  type GracePeriodMs,
  type GracePeriodValue,
  gracePeriodMsFromValue,
  isGracePeriodValue,
} from "./rotate-key.constants";

const COPY = {
  key: {
    noun: "Key",
    description: "Generate a fresh key while preserving this key's permissions and limits",
  },
  "root key": {
    noun: "Root Key",
    description: "Generate a fresh root key while preserving this root key's permissions",
  },
} as const;

type RotateInput = { keyId: string; expiration: GracePeriodMs };

type RotateMutation = {
  mutateAsync: (input: RotateInput) => Promise<{ key: string }>;
};

export type RotateKeyDialogProps = {
  /** The id of the key being rotated. */
  keyId: string;
  /** Names the key in the dialog title. */
  keyName: string | null;
  /** Mutation that performs the rotation and surfaces its own error toasts. */
  mutation: RotateMutation;
  /**
   * Runs once the user closes the success dialog. Use this to invalidate any
   * cached lists. Invalidating mid-flow remounts the triggering row and wipes
   * the success dialog before the user can copy the secret.
   */
  onRotated?: () => void;
  /** "key" or "root key". Drives all user-facing copy. */
  resourceLabel: "key" | "root key";
} & ActionComponentProps;

export function RotateKeyDialog({
  keyId,
  keyName,
  mutation,
  onRotated,
  resourceLabel,
  isOpen,
  onClose,
}: RotateKeyDialogProps) {
  const [gracePeriod, setGracePeriod] = useState<GracePeriodValue>(DEFAULT_GRACE_PERIOD);
  const [isConfirmed, setIsConfirmed] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [secret, setSecret] = useState<string | null>(null);

  const copy = COPY[resourceLabel];
  const title = keyName ? `Rotate “${keyName}”?` : `Rotate this ${resourceLabel}?`;

  const rotate = async () => {
    try {
      setIsLoading(true);
      const result = await mutation.mutateAsync({
        keyId,
        expiration: gracePeriodMsFromValue(gracePeriod),
      });
      setSecret(result.key);
    } catch {
      // The mutation hook surfaces its own toast.
    } finally {
      setIsLoading(false);
    }
  };

  const finish = () => {
    setSecret(null);
    onRotated?.();
    onClose();
  };

  if (secret) {
    return (
      <SecretKeyDialog
        secret={secret}
        title={`${copy.noun} rotated`}
        resourceLabel={copy.noun}
        onDone={finish}
      />
    );
  }

  return (
    <Dialog
      open={isOpen}
      onOpenChange={(open) => {
        if (!open && !isLoading) {
          onClose();
        }
      }}
    >
      <DialogContent className="w-full max-w-[560px] gap-4 rounded-2xl! border-grayA-4 p-6">
        <div className="flex flex-col gap-1">
          <DialogTitle>{title}</DialogTitle>
          <p className="text-[13px] leading-5 text-gray-11">{copy.description}</p>
        </div>
        <FormSelect
          label="Grace period"
          description={`How long the current ${resourceLabel} stays valid after rotation.`}
          options={GRACE_PERIOD_OPTIONS}
          value={gracePeriod}
          onValueChange={(value) => {
            if (isGracePeriodValue(value)) {
              setGracePeriod(value);
            }
          }}
        />
        <FormCheckbox
          size="lg"
          checked={isConfirmed}
          onCheckedChange={(next) => setIsConfirmed(next === true)}
          label={`I understand this will generate a new ${resourceLabel} and revoke the current one.`}
        />
        <div className="flex items-center justify-end gap-2 pt-1">
          <Button type="button" variant="outline" size="md" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button
            type="button"
            variant="primary"
            size="md"
            onClick={rotate}
            disabled={!isConfirmed || isLoading}
            loading={isLoading}
          >
            Rotate key
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
