import { RefreshCw } from "lucide-react";
import { useState } from "react";
import type { Key, RerollKeyResult } from "~/components/keys-table/schema/keys.schema";
import { Button } from "~/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "~/components/ui/dialog";
import { Field, FieldDescription, FieldError, FieldLabel } from "~/components/ui/field";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "~/components/ui/select";
import { DiscardSecretConfirm, SecretRevealCard, useSecretCloseGate } from "./secret-reveal-card";

const GRACE_PERIODS = [
  { value: "0", label: "Revoke immediately" },
  { value: "60000", label: "1 minute" },
  { value: "900000", label: "15 minutes" },
  { value: "3600000", label: "1 hour" },
  { value: "21600000", label: "6 hours" },
  { value: "86400000", label: "24 hours" },
] as const;

const DEFAULT_GRACE = "60000";

/** Performs the reroll and resolves with the one-time secret. */
export type RerollFn = (input: { keyId: string; expiration: number }) => Promise<RerollKeyResult>;

type RotateKeyDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  keyToRotate: Key | null;
  onReroll: RerollFn;
  onRerolled?: (result: RerollKeyResult) => void;
};

export function RotateKeyDialog({
  open,
  onOpenChange,
  keyToRotate,
  onReroll,
  onRerolled,
}: RotateKeyDialogProps) {
  const [grace, setGrace] = useState<string>(DEFAULT_GRACE);
  const [rotated, setRotated] = useState<RerollKeyResult | null>(null);
  const [hasCopied, setHasCopied] = useState(false);
  const [isRerolling, setIsRerolling] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const close = () => {
    if (rotated) {
      onRerolled?.(rotated);
    }
    setGrace(DEFAULT_GRACE);
    setRotated(null);
    setHasCopied(false);
    setIsRerolling(false);
    setError(null);
    onOpenChange(false);
  };

  const { tryClose, discardConfirm } = useSecretCloseGate({
    hasSecret: rotated !== null,
    hasCopied,
    onClose: close,
  });

  const handleRotate = async () => {
    if (!keyToRotate || isRerolling) {
      return;
    }
    setError(null);
    setIsRerolling(true);
    try {
      const result = await onReroll({ keyId: keyToRotate.id, expiration: Number(grace) });
      setRotated(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to rotate key. Please try again.");
    } finally {
      setIsRerolling(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(next) => (next ? onOpenChange(true) : tryClose())}>
      {keyToRotate ? (
        <DialogContent>
          {rotated === null ? (
            <ConfigureCard
              grace={grace}
              onGraceChange={setGrace}
              onCancel={tryClose}
              onRotate={handleRotate}
              isRerolling={isRerolling}
              error={error}
            />
          ) : (
            <SecretRevealCard
              title="Key rotated"
              description="A new secret has been generated. Copy it before closing."
              secretLabel="New secret"
              plaintext={rotated.plaintext}
              onCopied={() => setHasCopied(true)}
              onDone={tryClose}
            />
          )}
          {/* Rendered inside the parent dialog's popup so Base UI treats it
              as a nested dialog: Escape closes only the confirmation and the
              two open states don't race. */}
          <DiscardSecretConfirm {...discardConfirm} />
        </DialogContent>
      ) : null}
    </Dialog>
  );
}

type ConfigureCardProps = {
  grace: string;
  onGraceChange: (value: string) => void;
  onCancel: () => void;
  onRotate: () => void;
  isRerolling: boolean;
  error: string | null;
};

function ConfigureCard({
  grace,
  onGraceChange,
  onCancel,
  onRotate,
  isRerolling,
  error,
}: ConfigureCardProps) {
  return (
    <>
      <DialogHeader className="border-b-0 pb-2">
        <DialogTitle>Rotate key</DialogTitle>
        <DialogDescription>
          Generates a new secret while preserving this key's configuration.
        </DialogDescription>
      </DialogHeader>

      <DialogBody className="px-5 pt-2 pb-5">
        <Field data-invalid={!!error}>
          <FieldLabel htmlFor="rotate-key-grace">Grace period</FieldLabel>
          <Select
            items={GRACE_PERIODS.map((p) => ({ value: p.value, label: p.label }))}
            value={grace}
            onValueChange={(value) => value !== null && onGraceChange(value)}
            disabled={isRerolling}
          >
            <SelectTrigger id="rotate-key-grace">
              <SelectValue placeholder="Select a grace period" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {GRACE_PERIODS.map((p) => (
                  <SelectItem key={p.value} value={p.value}>
                    {p.label}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          {error ? (
            <FieldError>{error}</FieldError>
          ) : (
            <FieldDescription>
              How long the current key stays valid after rotation.
            </FieldDescription>
          )}
        </Field>
      </DialogBody>

      <DialogFooter>
        <Button type="button" variant="ghost" onClick={onCancel} disabled={isRerolling}>
          Cancel
        </Button>
        <Button type="button" onClick={onRotate} disabled={isRerolling}>
          <RefreshCw className={isRerolling ? "animate-spin" : undefined} />
          {isRerolling ? "Rotating…" : "Rotate key"}
        </Button>
      </DialogFooter>
    </>
  );
}
