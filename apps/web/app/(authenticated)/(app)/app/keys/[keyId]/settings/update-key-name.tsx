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
import { toast } from "@/components/ui/toaster";
import { cn } from "@/lib/utils";
import { updateKeyName } from "./actions";
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    name: string | null;
  };
};

export const UpdateKeyName: React.FC<Props> = ({ apiKey }) => {
  return (
    <form
      action={async (formData: FormData) => {
        const res = await updateKeyName(formData);
        if (res.error) {
          toast.error(res.error.message);
          return;
        }

        toast.success("Name has been updated");
      }}
    >
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
