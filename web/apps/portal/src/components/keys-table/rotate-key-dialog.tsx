import { RefreshCw } from "lucide-react";
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
import type { RotateKeyController } from "~/hooks/use-rotate-key";
import { useSecretCloseGate } from "~/hooks/use-secret-close-gate";
import { DiscardSecretConfirm, SecretRevealCard } from "./secret-reveal-card";

const GRACE_PERIODS = [
  { value: "0", label: "Revoke immediately" },
  { value: "60000", label: "1 minute" },
  { value: "900000", label: "15 minutes" },
  { value: "3600000", label: "1 hour" },
  { value: "21600000", label: "6 hours" },
  { value: "86400000", label: "24 hours" },
] as const;

function rotateErrorMessage(error: unknown): string | null {
  if (!error) {
    return null;
  }
  return error instanceof Error ? error.message : "Failed to rotate key. Please try again.";
}

type RotateKeyDialogProps = {
  rotate: RotateKeyController;
};

export function RotateKeyDialog({ rotate }: RotateKeyDialogProps) {
  const keyToRotate = rotate.rotating;
  const rotated = rotate.data ?? null;
  const isRerolling = rotate.isPending;
  const error = rotateErrorMessage(rotate.error);

  const { tryClose, discardConfirm } = useSecretCloseGate({
    hasSecret: rotated !== null,
    hasCopied: rotate.hasCopied,
    onClose: rotate.close,
  });

  return (
    <Dialog
      open={keyToRotate !== null}
      onOpenChange={(next) => {
        if (!next) {
          tryClose();
        }
      }}
    >
      {keyToRotate ? (
        <DialogContent>
          {rotated === null ? (
            <ConfigureCard
              grace={rotate.grace}
              onGraceChange={rotate.setGrace}
              onCancel={tryClose}
              onRotate={rotate.rotate}
              isRerolling={isRerolling}
              error={error}
            />
          ) : (
            <SecretRevealCard
              title="Key rotated"
              description="A new secret has been generated. Copy it before closing."
              secretLabel="New secret"
              plaintext={rotated.plaintext}
              onCopied={rotate.markCopied}
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
