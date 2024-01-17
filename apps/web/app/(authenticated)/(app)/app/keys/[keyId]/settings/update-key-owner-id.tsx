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
import { FormField } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import React, { useState } from "react";
import { Form, useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  keyId: z.string(),
  ownerId: z.string(),
});

type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    ownerId: string | null;
  };
};

export const UpdateKeyOwnerId: React.FC<Props> = ({ apiKey }) => {
  const router = useRouter();
  const [_isLoading, _setIsLoading] = useState(false);
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const updateOwnerId = trpc.keySettings.updateOwnerId.useMutation({
    onSuccess() {
      toast.success("Your owner ID has been updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    updateOwnerId.mutate(values);
  }
  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <Card>
          <CardHeader>
            <CardTitle>Owner ID</CardTitle>
            <CardDescription>
              Use this to identify the owner of the key. For example by setting the userId of the
              user in your system.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex justify-between item-center">
            <div className={cn("flex flex-col space-y-2 w-full ")}>
              <input type="hidden" name="keyId" value={apiKey.id} />
              <Label htmlFor="ownerId">OwnerId</Label>
              <FormField
                control={form.control}
                name="ownerId"
                render={({ field }) => (
                  <Input
                    {...field}
                    type="string"
                    className="h-8 max-w-sm"
                    defaultValue={apiKey.ownerId ?? ""}
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
