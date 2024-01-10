"use client";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
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
import { trpc } from "@/lib/trpc";

import React from "react";
import { useFormStatus } from "react-dom";

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

  function handleSubmit(event: any) {
    event.preventDefault();
    const formData = new FormData(event.target);
    const apiName = formData.get("name");
    const apiId = formData.get("apiId");
    const workspaceId = formData.get("workspaceId");

    const _updateName = trpc.apiSettings.updateName.mutate({
      name: apiName as string,
      apiId: apiId as string,
      workspaceId: workspaceId as string,
    });
    // .then((response) => {
    //   if (response) {
    //     // revalidatePath(`/apps/api/${apiId}`);
    //     toast({
    //       title: "Success",
    //       description: "Your email has been added to our newsletter!",
    //     });
    //   } else {
    //     toast({
    //       title: "Error",
    //       description: "Something went wrong. Please try again later",
    //     });
    //   }
    // });
  }
  return (
    <form onSubmit={handleSubmit}>
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
            <label className="sr-only hidden">Name</label>
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
