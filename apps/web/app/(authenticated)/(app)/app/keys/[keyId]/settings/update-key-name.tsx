"use client";
import React, { useState } from "react";

import { SubmitButton } from "@/components/dashboard/submit-button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { FormField } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { Form, useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  keyId: z.string(),
  name: z.string(),
});
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    name: string | null;
  };
};

export const UpdateKeyName: React.FC<Props> = ({ apiKey }) => {
  const router = useRouter();
  const [isLoading, _setIsLoading] = useState(false);

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const updateName = trpc.keySettings.updateName.useMutation({
    onSuccess() {
      toast.success("Your key name uses has been updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    updateName.mutate(values);
  }
  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
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
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <Input
                    type="string"
                    className="h-8 max-w-sm"
                    defaultValue={apiKey.name ?? ""}
                    autoComplete="off"
                  />
                )}
              />
            </div>
          </CardContent>
          <CardFooter className="justify-end">
            <SubmitButton label="Save" />
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
