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
import { cn } from "@/lib/utils";
import React from "react";
import { updateKeyOwnerId } from "./actions";
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    ownerId: string | null;
  };
};

export const UpdateKeyOwnerId: React.FC<Props> = ({ apiKey }) => {
  return (
    <form
      action={async (formData: FormData) => {
        const res = await updateKeyOwnerId(formData);
        if (res.error) {
          toast.error(res.error.message);
          return;
        }

        toast.success("Owner ID has been updated");
      }}
    >
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
