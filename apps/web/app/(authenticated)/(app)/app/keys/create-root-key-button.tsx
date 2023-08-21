"use client";

import { useToast } from "@/components/ui/use-toast";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { CopyButton } from "@/components/dashboard/copy-button";
import { VisibleButton } from "@/components/dashboard/visible-button";
import { Loading } from "@/components/dashboard/loading";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Code } from "@/components/ui/code";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { trpc } from "@/lib/trpc/client";
import { AlertTriangle } from "lucide-react";

type Props = {
  apiId?: string;
};

export const CreateRootKeyButton: React.FC<Props> = ({ apiId }) => {
  const { toast } = useToast();

  const router = useRouter();

  const key = trpc.key.createInternalRootKey.useMutation({
    onSuccess() {
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast({
        title: "Error",
        description: err.message,
        variant: "destructive",
      });
    },
  });

  const snippet = `curl -XPOST '${process.env.NEXT_PUBLIC_UNKEY_API_URL ?? "https://api.unkey.dev"}/v1/keys' \\
  -H 'Authorization: Bearer ${key.data?.key}' \\
  -H 'Content-Type: application/json' \\
  -d '{
    "prefix": "hello",
    "apiId": "${apiId ?? "<API_ID>"}"
  }'
  `;

  const maskedKey = `unkey_${"*".repeat(key.data?.key.split("_").at(1)?.length ?? 0)}`;
  const [showKey, setShowKey] = useState(false);
  const [showKeyInSnippet, setShowKeyInSnippet] = useState(false);
  return (
    <>
      <Dialog
        onOpenChange={(v) => {
          if (!v) {
            // Remove the key from memory when closing the modal
            key.reset();
            router.refresh();
          }
        }}
      >
        <DialogTrigger asChild>
          <Button>Create New Key</Button>
        </DialogTrigger>

        {key.data ? (
          <DialogContent className="max-w-fit ">
            <DialogHeader>
              <DialogTitle>Your API Key</DialogTitle>
              <DialogDescription>
                This key is only shown once and can not be recovered. Please store it somewhere
                safe.
              </DialogDescription>
              <div>
                <Alert variant="destructive" className="my-4">
                  <AlertTriangle className="w-4 h-4" />
                  <AlertTitle>Root Key Generated</AlertTitle>
                  <AlertDescription>
                    The root key will provide full read and write access to all current and future
                    resources.
                    <br />
                    For production use, we recommend creating a key with only the permissions you
                    need.
                  </AlertDescription>
                </Alert>
              </div>

              <Code className="flex items-center justify-between w-full gap-4 my-8 ">
                {showKey ? key.data.key : maskedKey}
                <div className="flex items-start justify-between gap-4">
                  <VisibleButton isVisible={showKey} setIsVisible={setShowKey} />
                  <CopyButton value={key.data.key} />
                </div>
              </Code>
            </DialogHeader>

            <p className="mt-2 text-sm font-medium text-center text-stone-700 ">
              Try creating a new api key for your users:
            </p>
            <Code className="flex items-start justify-between w-full gap-4 my-8 ">
              {showKeyInSnippet ? snippet : snippet.replace(key.data.key, maskedKey)}
              <div className="flex items-start justify-between gap-4">
                <VisibleButton isVisible={showKeyInSnippet} setIsVisible={setShowKeyInSnippet} />
                <CopyButton value={snippet} />
              </div>
            </Code>
          </DialogContent>
        ) : (
          <DialogContent>
            <DialogTitle>Create a new API key</DialogTitle>
            <DialogDescription />

            <Alert id="root-key-alert">
              <AlertTriangle className="w-4 h-4" />
              <AlertTitle id="root-key-alert-title">Root keys can be dangerous</AlertTitle>
              <AlertDescription>
                The root key will provide full read and write access to all current and future
                resources.
                <br />
                For production use, we recommend creating a key with only the permissions you need.
              </AlertDescription>
            </Alert>
            <DialogFooter className="flex items-center justify-between gap-2 ">
              <Button disabled={key.isLoading} onClick={() => key.mutate()}>
                {key.isLoading ? <Loading /> : "Create Root Key"}
              </Button>
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </>
  );
};
