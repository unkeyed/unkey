"use client";
import { revalidate } from "@/app/actions";
import { Card, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { toast } from "@/components/ui/toaster";
import { Button } from "@unkey/ui";
import type React from "react";
import { useState } from "react";

import { Loading } from "@/components/dashboard/loading";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { trpc } from "@/lib/trpc/client";
import { useRouter } from "next/navigation";

type Props = {
  apiKey: {
    id: string;
  };
  keyAuthId: string;
};

export const DeleteKey: React.FC<Props> = ({ apiKey, keyAuthId }) => {
  const router = useRouter();
  const [open, setOpen] = useState(false);

  const deleteKey = trpc.key.delete.useMutation({
    onSuccess() {
      revalidate(`/keys/${keyAuthId}/keys`);
      toast.success("Key deleted");
      router.push("/apis");
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  return (
    <>
      <Card className="relative border-alert">
        <CardHeader>
          <CardTitle>Delete</CardTitle>
          <CardDescription>This key will be deleted. This action cannot be undone.</CardDescription>
        </CardHeader>

        <CardFooter className="z-10 justify-end">
          <Button type="button" onClick={() => setOpen(!open)} variant="destructive">
            Delete Key
          </Button>
        </CardFooter>
      </Card>

      <Dialog open={open} onOpenChange={(o) => setOpen(o)}>
        <DialogContent className="border-alert">
          <DialogHeader>
            <DialogTitle>Delete Key</DialogTitle>
            <DialogDescription>
              This key will be deleted. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>

          <Alert variant="alert">
            <AlertTitle>Warning</AlertTitle>
            <AlertDescription>This action is not reversible. Please be certain.</AlertDescription>
          </Alert>
          <input type="hidden" name="keyId" value={apiKey.id} />

          <DialogFooter className="justify-end">
            <Button type="button" onClick={() => setOpen(!open)}>
              Cancel
            </Button>
            <Button
              type="submit"
              variant="destructive"
              disabled={deleteKey.isLoading}
              onClick={() => deleteKey.mutate({ keyIds: [apiKey.id] })}
            >
              {deleteKey.isLoading ? <Loading /> : "Delete Key"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};
