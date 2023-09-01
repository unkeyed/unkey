"use client";
import { Button } from "@/components/ui/button";
import React, { useState } from "react";
import { experimental_useFormStatus as useFormStatus } from "react-dom";

import { Card, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";
import { deleterApi } from "./actions";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

type Props = {
  api: {
    id: string;
    workspaceId: string;
    name: string;
  };
};

export const DeleteApi: React.FC<Props> = ({ api }) => {
  const { toast } = useToast();
  const { pending } = useFormStatus();

  const [open, setOpen] = useState(false);
  const [confirmName, setConfirmName] = useState("");
  const [confirmIntent, setConfirmIntent] = useState("");

  const isValid = confirmIntent === "delete my api" && confirmName === api.name;

  return (
    <>
      <Card className="relative border-alert">
        <CardHeader>
          <CardTitle>Delete</CardTitle>
          <CardDescription>
            This api will be deleted, along with all of its keys and data. This action cannot be
            undone.
          </CardDescription>
        </CardHeader>

        <CardFooter className="z-10 justify-end">
          <Button type="button" onClick={() => setOpen(!open)} variant="alert">
            Delete API
          </Button>
        </CardFooter>
      </Card>
      <Dialog open={open} onOpenChange={(o) => setOpen(o)}>
        <form
          action={async (formData: FormData) => {
            const res = await deleterApi(formData);
            if (res.error) {
              toast({
                title: "Error",
                description: res.error,
                variant: "alert",
              });
              return;
            }
            toast({
              title: "Success",
              description: "Api name updated",
            });
          }}
        >
          <DialogContent className="border-alert">
            <DialogHeader>
              <DialogTitle>Delete API</DialogTitle>
              <DialogDescription>
                This api will be deleted, along with all of its keys. This action cannot be undone.
              </DialogDescription>
            </DialogHeader>

            <Alert variant="alert">
              <AlertTitle>Warning</AlertTitle>
              <AlertDescription>This action is not reversible. Please be certain.</AlertDescription>
            </Alert>

            <input type="hidden" name="workspaceId" value={api.workspaceId} />
            <input type="hidden" name="apiId" value={api.id} />
            <div className="flex flex-col space-y-2">
              <label className="text-sm text-content-subtle">
                Enter the API name <span className="font-medium text-content">{api.name}</span> to
                continue:
              </label>
              <Input
                name="name"
                value={confirmName}
                onChange={(v) => setConfirmName(v.currentTarget.value)}
                autoComplete="off"
              />
            </div>
            <div className="flex flex-col space-y-2">
              <label className="text-sm text-content-subtle">
                To verify, type <span className="font-medium text-content">delete my api</span>{" "}
                below:
              </label>
              <Input
                name="intent"
                value={confirmIntent}
                onChange={(v) => setConfirmIntent(v.currentTarget.value)}
                autoComplete="off"
              />
            </div>

            <DialogFooter className="justify-end">
              <Button type="button" onClick={() => setOpen(!open)} variant="secondary">
                Cancel
              </Button>
              <Button
                type="submit"
                variant={isValid ? "alert" : "disabled"}
                disabled={!isValid || pending}
              >
                Delete API
              </Button>
            </DialogFooter>
          </DialogContent>
        </form>
      </Dialog>
    </>
  );
};
