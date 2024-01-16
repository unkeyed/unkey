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
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { useRouter } from "next/navigation";

import { FormField } from "@/components/ui/form";
import { zodResolver } from "@hookform/resolvers/zod";
import React, { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
const formSchema = z.object({
  name: z.string(),
  apiId: z.string(),
  workspaceId: z.string(),
});

type Props = {
  api: {
    id: string;
    workspaceId: string;
    name: string;
  };
};

export const UpdateApiName: React.FC<Props> = ({ api }) => {
  const updateName = trpc.api.updateName.useMutation();
  const [isLoading, setLoading] = useState(false);

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: api.name,
      apiId: api.id,
      workspaceId: api.workspaceId,
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    try {
      setLoading(true);
      await updateName.mutate({
        name: values.name,
        apiId: values.apiId,
        workspaceId: values.workspaceId,
      });
      toast({
        title: "Success",
        description: "Your Api name has been updated!",
      });
      router.refresh();
    } catch (err) {
      toast({
        title: "Error",
        description: (err as Error).message,
        variant: "alert",
      });
    } finally {
      setLoading(false);
    }
  }
  const router = useRouter();
  return (
    <form onSubmit={form.handleSubmit(onSubmit)}>
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
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => <Input className="max-w-sm" {...field} autoComplete="off" />}
            />
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button
            variant={form.formState.isValid && !isLoading ? "primary" : "disabled"}
            disabled={!form.formState.isValid || isLoading}
            type="submit"
          >
            {isLoading ? <Loading /> : "Save"}
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
};
