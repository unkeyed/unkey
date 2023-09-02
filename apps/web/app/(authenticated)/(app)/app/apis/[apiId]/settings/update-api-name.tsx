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
import { useToast } from "@/components/ui/use-toast";
import { updateApiName } from "./actions";
type Props = {
  api: {
    id: string;
    workspaceId: string;
    name: string;
  };
};

export const UpdateApiName: React.FC<Props> = ({ api }) => {
  const { toast } = useToast();
  const { pending } = useFormStatus();

  return (
    <form
      action={async (formData: FormData) => {
        const res = await updateApiName(formData);
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
      <Card>
        <CardHeader>
          <CardTitle>Api Name</CardTitle>
          <CardDescription>
            Api names are not customer facing. Choose a name that makes it easy to recognize for
            you.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col space-y-2">
            <input type="hidden" name="workspaceId" value={api.workspaceId} />
            <input type="hidden" name="apiId" value={api.id} />
            <label className="hidden sr-only">Name</label>
            <Input name="name" className="max-w-sm" defaultValue={api.name} autoComplete="off" />
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button variant={pending ? "disabled" : "primary"} type="submit" disabled={pending}>
            {pending ? <Loading /> : "Save"}
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
};
