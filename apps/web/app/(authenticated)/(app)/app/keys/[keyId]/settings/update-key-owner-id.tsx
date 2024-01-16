"use client";
import { SubmitButton } from "@/components/dashboard/submit-button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc";
import { cn } from "@/lib/utils";
import React from "react";

type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    ownerId: string | null;
  };
};

export const UpdateKeyOwnerId: React.FC<Props> = ({ apiKey }) => {
  const { toast } = useToast();

  function handleSubmit(event: any) {
    event.preventDefault();
    const formData = new FormData(event.target);
    const keyId = event.target.keyId.value;

    const _updateOwnerId = trpc.keySettings.updateOwnerId
      .mutate({
        keyId: keyId as string,
        ownerId: formData.get("ownerId") as string,
      })
      .then((response) => {
        if (response) {
          toast({
            title: "Success",
            description: "Your owner ID has been updated!",
          });
        } else {
          toast({
            title: "Error",
            description: "Something went wrong. Please try again later",
          });
        }
      });
  }
  return (
    <form onSubmit={handleSubmit}>
      <Card>
        <CardHeader>
          <CardTitle>Owner ID</CardTitle>
          <CardDescription>
            Use this to identify the owner of the key. For example by setting the userId of the user
            in your system.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex justify-between item-center">
          <div className={cn("flex flex-col space-y-2 w-full ")}>
            <input type="hidden" name="keyId" value={apiKey.id} />
            <Label htmlFor="ownerId">OwnerId</Label>
            <Input
              type="string"
              name="ownerId"
              className="h-8 max-w-sm"
              defaultValue={apiKey.ownerId ?? ""}
              autoComplete="off"
            />
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <SubmitButton label="Save" />
        </CardFooter>
      </Card>
    </form>
  );
};
