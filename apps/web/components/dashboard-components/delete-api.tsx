"use client";

import { useToast } from "@/components/ui/use-toast";
import { useRouter } from "next/navigation";

import { Loading } from "./loading";
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
  apiId: string;
  apiName: string;
};
export const DeleteApiButton: React.FC<Props> = ({ apiId, apiName }) => {
  const { toast } = useToast();
  const [typedName, setTypedName] = useState("");

  const router = useRouter();
  const deleteApi = trpc.api.delete.useMutation({
    onSuccess() {
      toast({
        title: "API deleted",
        description: "Your API and all its keys have been removed",
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
        <Button variant="destructive">Delete API</Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Are you sure?</DialogTitle>
          <DialogDescription>
            {" "}
            Do you really want to delete this API and all its keys?
          </DialogDescription>
        </DialogHeader>
        If you want to continue, please type in the name of the api below:{" "}
        <Code className="mt-2 text-center bg-gray-100 rounded ">{apiName}</Code>

        <Input
          value={typedName}
          placeholder={apiName}
          onChange={(v) => setTypedName(v.currentTarget.value)}
        />

        <DialogFooter>
          <Button
            disabled={typedName !== apiName || deleteApi.isLoading}
            variant="destructive"
            onClick={() => {
              if (typedName === apiName) {
                deleteApi.mutate({ apiId });
              }
            }}
          >
            {deleteApi.isLoading ? <Loading /> : "Delete"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
