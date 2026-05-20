import { AlertTriangle, Check, Copy, Eye, EyeOff } from "lucide-react";
import { type ReactNode, useEffect, useRef, useState } from "react";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "~/components/ui/alert-dialog";
import { Button } from "~/components/ui/button";
import {
  DialogBody,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "~/components/ui/dialog";
import { Tooltip, TooltipContent, TooltipTrigger } from "~/components/ui/tooltip";

type SecretRevealCardProps = {
  title: string;
  description: string;
  secretLabel?: string;
  plaintext: string;
  trailer?: ReactNode;
  onCopied: () => void;
  onDone: () => void;
};

export function SecretRevealCard({
  title,
  description,
  secretLabel = "Secret key",
  plaintext,
  trailer,
  onCopied,
  onDone,
}: SecretRevealCardProps) {
  const [revealed, setRevealed] = useState(false);
  const [justCopied, setJustCopied] = useState(false);
  const justCopiedTimer = useRef<number | null>(null);

  const masked = "•".repeat(Math.max(plaintext.length, 24));

  useEffect(() => {
    return () => {
      if (justCopiedTimer.current !== null) {
        window.clearTimeout(justCopiedTimer.current);
      }
    };
  }, []);

  const handleCopy = () => {
    navigator.clipboard.writeText(plaintext).catch(() => {});
    onCopied();
    setJustCopied(true);
    if (justCopiedTimer.current !== null) {
      window.clearTimeout(justCopiedTimer.current);
    }
    justCopiedTimer.current = window.setTimeout(() => {
      setJustCopied(false);
      justCopiedTimer.current = null;
    }, 1500);
  };

  return (
    <>
      <DialogHeader className="border-b-0 pb-2">
        <DialogTitle>{title}</DialogTitle>
        <DialogDescription className="sr-only">{description}</DialogDescription>
      </DialogHeader>

      <DialogBody className="px-5 pt-2 pb-5">
        <Alert variant="warning">
          <AlertTriangle />
          <AlertTitle>Save this key now</AlertTitle>
          <AlertDescription>
            This is the only time we'll show the full secret. You won't be able to view it again.
          </AlertDescription>
        </Alert>

        <div className="mt-5 flex flex-col gap-1.5">
          <span className="font-medium text-gray-12 text-sm">{secretLabel}</span>

          <div className="flex items-center gap-1.5 rounded-md border border-primary/15 bg-gray-2 py-1 pr-1 pl-3">
            <span className="min-w-0 flex-1 truncate font-mono text-gray-12 text-sm">
              {revealed ? plaintext : masked}
            </span>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  aria-label={revealed ? "Hide secret" : "Reveal secret"}
                  onClick={() => setRevealed((v) => !v)}
                  className="size-7 text-gray-11 shadow-none hover:bg-gray-3 [&_svg]:size-3.5"
                >
                  {revealed ? <EyeOff /> : <Eye />}
                </Button>
              </TooltipTrigger>
              <TooltipContent>{revealed ? "Hide secret" : "Reveal secret"}</TooltipContent>
            </Tooltip>

            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleCopy}
              autoFocus
              className="h-7 min-w-[5.25rem] px-2.5 [&_svg]:size-3.5"
            >
              <span
                key={justCopied ? "copied" : "copy"}
                className="motion-safe:fade-in motion-safe:slide-in-from-bottom-1 inline-flex items-center gap-1.5 motion-safe:animate-in motion-safe:duration-150"
              >
                {justCopied ? (
                  <>
                    <Check />
                    Copied
                  </>
                ) : (
                  <>
                    <Copy />
                    Copy
                  </>
                )}
              </span>
            </Button>
          </div>

          {trailer ? <span className="text-gray-11 text-xs">{trailer}</span> : null}
        </div>
      </DialogBody>

      <DialogFooter>
        <Button onClick={onDone}>Done</Button>
      </DialogFooter>
    </>
  );
}

type DiscardSecretConfirmProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDiscard: () => void;
};

export function DiscardSecretConfirm({ open, onOpenChange, onDiscard }: DiscardSecretConfirmProps) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>You won't see this secret key again</AlertDialogTitle>
          <AlertDialogDescription>
            Make sure to copy your secret key before closing. It cannot be retrieved later.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={onDiscard}>
            Close anyway
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

type CloseGateOptions = {
  hasSecret: boolean;
  hasCopied: boolean;
  onClose: () => void;
};

type CloseGate = {
  tryClose: () => void;
  discardConfirm: {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onDiscard: () => void;
  };
};

/**
 * Gates dialog dismissal when a freshly-revealed secret hasn't been copied yet.
 * Routes ESC, outside-click, and the X button through `tryClose`; pops a
 * confirmation AlertDialog if the user is about to lose the only chance to
 * grab the secret.
 */
export function useSecretCloseGate({ hasSecret, hasCopied, onClose }: CloseGateOptions): CloseGate {
  const [alertOpen, setAlertOpen] = useState(false);
  const requiresConfirm = hasSecret && !hasCopied;

  const tryClose = () => {
    if (requiresConfirm) {
      setAlertOpen(true);
    } else {
      onClose();
    }
  };

  const onDiscard = () => {
    setAlertOpen(false);
    onClose();
  };

  return {
    tryClose,
    discardConfirm: { open: alertOpen, onOpenChange: setAlertOpen, onDiscard },
  };
}
