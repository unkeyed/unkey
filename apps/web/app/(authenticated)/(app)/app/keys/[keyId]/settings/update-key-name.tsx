"use client";
import React from "react";

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
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc";
import { cn } from "@/lib/utils";

type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    name: string | null;
  };
};

export const UpdateKeyName: React.FC<Props> = ({ apiKey }) => {
  const { toast } = useToast();
  function handleSubmit(event: any) {
    event.preventDefault();
    const formData = new FormData(event.target);
    const keyId = event.target.keyId.value;

    const _updateName = trpc.keySettings.updateName
      .mutate({
        keyId: keyId as string,
        name: formData.get("name") as string,
      })
      .then((response) => {
        if (response) {
          toast({
            title: "Success",
            description: "Your key name uses has been updated!",
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
          <CardTitle>Name</CardTitle>
          <CardDescription>
            To make it easier to identify a particular key, you can provide a name.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex justify-between item-center">
          <div className={cn("flex flex-col space-y-2 w-full ")}>
            <input type="hidden" name="keyId" value={apiKey.id} />
            <Label htmlFor="remaining">Name</Label>
            <Input
              type="string"
              name="name"
              className="h-8 max-w-sm"
              defaultValue={apiKey.name ?? ""}
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
