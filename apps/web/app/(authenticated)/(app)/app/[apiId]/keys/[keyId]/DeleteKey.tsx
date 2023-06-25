"use client";

import { useToast } from "@/components/ui/use-toast";
import { useRouter } from "next/navigation";

import { Loading } from "@/components/loading";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { trpc } from "@/lib/trpc/client";
import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Code } from "@/components/ui/code";

type Props = {
  keyId: string;
  keyStart: string;
};
export const DeleteKeyButton: React.FC<Props> = ({ keyId, keyStart }) => {
  const { toast } = useToast();

  const router = useRouter();
  const deleteKey = trpc.key.delete.useMutation({
    onSuccess() {
      toast({
        title: "Key deleted",
        description: "Your Key has been removed",
      });
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast({ title: "Error", description: err.message, variant: "destructive" });
    },
  });

  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="destructive">Delete Key</Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Are you sure?</DialogTitle>
          <DialogDescription> Do you really want to delete this key?</DialogDescription>
        </DialogHeader>

        <Code className="mt-2 text-center bg-gray-100 rounded ">{keyStart}</Code>

        <DialogFooter>
          <Button
            disabled={deleteKey.isLoading}
            variant="destructive"
            onClick={() => {
              deleteKey.mutate({ keyIds: [keyId] });
            }}
          >
            {deleteKey.isLoading ? <Loading /> : "Delete"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
