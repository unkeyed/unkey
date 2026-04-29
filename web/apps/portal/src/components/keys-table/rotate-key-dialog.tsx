import { RefreshCw } from "lucide-react";
import { useState } from "react";
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
import { Field, FieldDescription, FieldLabel } from "~/components/ui/field";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "~/components/ui/select";
import { newPlaintext } from "~/lib/random-key";
import type { Key } from "~/routes/dave-initial-design/-seed";
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

export type RotateResult = {
  start: string;
};

type RotateKeyDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  keyToRotate: Key | null;
  onRotate: (id: string, result: RotateResult) => void;
};

export function RotateKeyDialog({
  open,
  onOpenChange,
  keyToRotate,
  onRotate,
}: RotateKeyDialogProps) {
  const [grace, setGrace] = useState<string>(DEFAULT_GRACE);
  const [rotated, setRotated] = useState<{ plaintext: string; start: string } | null>(null);
  const [hasCopied, setHasCopied] = useState(false);

  const close = () => {
    if (rotated && keyToRotate) {
      onRotate(keyToRotate.id, { start: rotated.start });
    }
    setGrace(DEFAULT_GRACE);
    setRotated(null);
    setHasCopied(false);
    onOpenChange(false);
  };

  const { tryClose, discardConfirm } = useSecretCloseGate({
    hasSecret: rotated !== null,
    hasCopied,
    onClose: close,
  });

  const handleRotate = () => {
    setRotated(newPlaintext());
  };

  return (
    <>
      <Dialog open={open} onOpenChange={(next) => (next ? onOpenChange(true) : tryClose())}>
        {keyToRotate ? (
          <DialogContent>
            {rotated === null ? (
              <ConfigureCard
                grace={grace}
                onGraceChange={setGrace}
                onCancel={tryClose}
                onRotate={handleRotate}
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
          </DialogContent>
        ) : null}
      </Dialog>

      <DiscardSecretConfirm {...discardConfirm} />
    </>
  );
}

type ConfigureCardProps = {
  grace: string;
  onGraceChange: (value: string) => void;
  onCancel: () => void;
  onRotate: () => void;
};

function ConfigureCard({ grace, onGraceChange, onCancel, onRotate }: ConfigureCardProps) {
  return (
    <>
      <DialogHeader className="border-b-0 pb-2">
        <DialogTitle>Rotate key</DialogTitle>
        <DialogDescription>
          Generates a new secret while preserving this key's configuration.
        </DialogDescription>
      </DialogHeader>

      <DialogBody className="px-5 pt-2 pb-5">
        <Field>
          <FieldLabel htmlFor="rotate-key-grace">Grace period</FieldLabel>
          <Select value={grace} onValueChange={onGraceChange}>
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
          <FieldDescription>How long the current key stays valid after rotation.</FieldDescription>
        </Field>
      </DialogBody>

      <DialogFooter>
        <Button type="button" variant="ghost" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="button" onClick={onRotate}>
          <RefreshCw />
          Rotate key
        </Button>
      </DialogFooter>
    </>
  );
}
