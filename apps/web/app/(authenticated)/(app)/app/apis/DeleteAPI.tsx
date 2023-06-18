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
  apiId: string;
};
export const DeleteAPIButton: React.FC<PropsWithChildren<Props>> = ({ apiId, children }) => {
  const deleteAPI = trpc.api.delete.useMutation();

  const { toast } = useToast();
  const router = useRouter();

  return (
    <>
      <Dialog>
        <DialogTrigger asChild>{children}</DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete API</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete this API? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="destructive"
              disabled={deleteAPI.isLoading}
              onClick={async () => {
                try {
                  await deleteAPI.mutateAsync({ apiId });

                  router.refresh();
                  toast({
                    title: "API deleted",
                  });
                } catch (e) {
                  toast({
                    title: "Error deleting API",
                    description: (e as Error).message,
                    variant: "destructive",
                  });
                }
              }}
            >
              {deleteAPI.isLoading ? <Loading /> : "Delete API"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};
