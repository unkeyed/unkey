"use client";

import React, { PropsWithChildren, useState } from "react";
import { useRouter } from "next/navigation";
import { useToast } from "@/components/ui/use-toast";

import { trpc } from "@/lib/trpc/client";
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

type Props = {
  keyId: string;
};
export const DeleteKeyButton: React.FC<PropsWithChildren<Props>> = ({ keyId, children }) => {
  const deleteKey = trpc.key.delete.useMutation();

  const { toast } = useToast();
  const router = useRouter();

  return (
    <>
      <Dialog>
        <DialogTrigger asChild>{children}</DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete API Key</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete this API key? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="destructive"
              disabled={deleteKey.isLoading}
              onClick={async () => {
                try {
                  await deleteKey.mutateAsync({ keyId });

                  router.refresh();
                  toast({
                    title: "Key deleted",
                  });
                } catch (e) {
                  toast({
                    title: "Error deleting key",
                    description: (e as Error).message,
                    variant: "destructive",
                  });
                }
              }}
            >
              {deleteKey.isLoading ? <Loading /> : "Delete Key"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};
