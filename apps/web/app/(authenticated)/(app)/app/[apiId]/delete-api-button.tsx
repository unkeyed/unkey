"use client";

import { useToast } from "@/components/ui/use-toast";
import { useRouter } from "next/navigation";
import { Loading } from "@/components/dashboard/loading";
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
      // replace to avoid back button to a delete API.
      router.replace("/app/apis");
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

  const [modalOpen, setModalOpen] = useState(false);

  return (
    <Dialog open={modalOpen} onOpenChange={setModalOpen}>
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
        <Code className="mt-2 text-center bg-gray-100 rounded dark:bg-stone-900 hover:border-stone-200 dark:hover:border-stone-800">
          {apiName}
        </Code>
        <form>
          <Input
            value={typedName}
            placeholder={apiName}
            onChange={(v) => setTypedName(v.currentTarget.value)}
          />
          <DialogFooter className="mt-4">
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
        </form>
      </DialogContent>
    </Dialog>
  );
};
