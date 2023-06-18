"use client";

import { useReducer, useState } from "react";
import { useRouter } from "next/navigation";
import { useToast } from "@/components/ui/use-toast";

import { trpc } from "@/lib/trpc/client";
import { CopyButton } from "@/components/CopyButton";
import { Loading } from "@/components/loading";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { AlertTriangle, Info } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";

type Props = {};

export const CreateKeyButton: React.FC<Props> = () => {
  const { toast } = useToast();

  const router = useRouter();

  const key = trpc.key.createInternalRootKey.useMutation({
    onError(err) {
      console.error(err);
      toast({
        title: "Error",
        description: err.message,
        variant: "destructive",
      });
    },
  });

  const snippet = "TODO: andreas";

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
        <DialogTrigger>
          <Button>Create New Key</Button>
        </DialogTrigger>

        {key.data ? (
          <DialogContent>
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

              <div className="flex items-center justify-between gap-4 px-2 py-1 mt-4 border rounded lg:p-4 border-white/10 bg-zinc-100 dark:bg-zinc-900">
                <pre className="font-mono">{key.data.key}</pre>
                <CopyButton value={key.data.key} />
              </div>
            </DialogHeader>

            <p className="mt-2 text-sm font-medium text-center text-zinc-100 ">
              Try it out with curl
            </p>
            <div className="flex items-start justify-between gap-4 px-2 py-1 border rounded lg:p-4 border-white/10 bg-zinc-100 dark:bg-zinc-900">
              <pre className="font-mono">{snippet}</pre>
              <CopyButton value={snippet} />
            </div>
          </DialogContent>
        ) : (
          <DialogContent>
            <DialogTitle>Create a new API key</DialogTitle>
            <DialogDescription />

            <Alert>
              <AlertTriangle className="w-4 h-4" />
              <AlertTitle>Root keys can be dangerous</AlertTitle>
              <AlertDescription>
                The root key will provide full read and write access to all current and future
                resources.
                <br />
                For production use, we recommend creating a key with only the permissions you need.
              </AlertDescription>
            </Alert>
            <DialogFooter className="flex items-center justify-between gap-2 ">
              <Button onClick={() => key.mutate()}>
                {key.isLoading ? <Loading /> : "Create Root Key"}
              </Button>
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </>
  );
};
