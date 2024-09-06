import { CopyButton } from "@/components/dashboard/copy-button";
import { VisibleButton } from "@/components/dashboard/visible-button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Code } from "@/components/ui/code";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { AlertCircle } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

type Props = {
  newKey:
    | {
        keyId: `key_${string}`;
        key: string;
      }
    | undefined;
  apiId: string;
  keyAuthId: string;
};

export function RerollNewKeyDialog({ newKey, apiId, keyAuthId }: Props) {
  if (!newKey) {
    return null;
  }

  const split = newKey.key.split("_") ?? [];
  const maskedKey =
    split.length >= 2
      ? `${split.at(0)}_${"*".repeat(split.at(1)?.length ?? 0)}`
      : "*".repeat(split.at(0)?.length ?? 0);
  const [showKey, setShowKey] = useState(false);

  const [open, setOpen] = useState(Boolean(newKey));

  return (
    <Dialog open={open} onOpenChange={(o: boolean) => setOpen(o)}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Your New API Key</DialogTitle>
        </DialogHeader>
        <Alert>
          <AlertCircle className="w-4 h-4" />
          <AlertTitle>This key is only shown once and can not be recovered </AlertTitle>
          <AlertDescription>
            Please pass it on to your user or store it somewhere safe.
          </AlertDescription>
        </Alert>
        <Code className="flex items-center justify-between w-full gap-4 my-8 ph-no-capture max-sm:text-xs sm:overflow-hidden">
          <pre>{showKey ? newKey.key : maskedKey}</pre>
          <div className="flex items-start justify-between gap-4 max-sm:absolute max-sm:right-11">
            <VisibleButton isVisible={showKey} setIsVisible={setShowKey} />
            <CopyButton value={newKey.key} />
          </div>
        </Code>
        <DialogFooter className="justify-end">
          <Link href={`/keys/${keyAuthId}`}>
            <Button variant="secondary">Back</Button>
          </Link>
          <Link href={`/apis/${apiId}/keys/${keyAuthId}/${newKey.keyId}`}>
            <Button variant="secondary">View key details</Button>
          </Link>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
