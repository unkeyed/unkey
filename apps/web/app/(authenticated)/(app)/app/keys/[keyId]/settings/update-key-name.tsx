"use client";
import { Button } from "@/components/ui/button";
import React from "react";
import { experimental_useFormStatus as useFormStatus } from "react-dom";

import { Loading } from "@/components/dashboard/loading";
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
  const { toast } = useToast();
  const { pending } = useFormStatus();
  return (
    <form
      action={async (formData: FormData) => {
        const res = await updateKeyName(formData);
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
          description: "Name has been updated",
        });
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
              className="max-w-sm h-8"
              defaultValue={apiKey.name ?? ""}
              autoComplete="off"
            />
          </div>
        </CardContent>
        <CardFooter className="justify-between">
          <Button variant={pending ? "disabled" : "primary"} type="submit" disabled={pending}>
            {pending ? <Loading /> : "Save"}
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
};
