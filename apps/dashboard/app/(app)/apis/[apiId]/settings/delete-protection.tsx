"use client";
import { Button } from "@unkey/ui";
import type React from "react";
import { useState } from "react";

import { Card, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";

import { Loading } from "@/components/dashboard/loading";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { revalidate } from "./actions";

type Props = {
  api: {
    id: string;
    workspaceId: string;
    name: string;
    deleteProtection: boolean | null;
  };
};

export const DeleteProtection: React.FC<Props> = ({ api }) => {
  const [open, setOpen] = useState(false);

  const formSchema = z.object({
    name: z.string().refine((v) => v === api.name, "Please confirm the API name"),
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const router = useRouter();

  const updateDeleteProtection = trpc.api.updateDeleteProtection.useMutation({
    async onSuccess(_, { enabled }) {
      toast.message(
        `Delete protection for ${api.name} has been ${enabled ? "enabled" : "disabled"}`,
        {},
      );

      setOpen(false);
      await revalidate();

      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  const isValid = form.watch("name") === api.name;

  async function onSubmit(_: z.infer<typeof formSchema>) {
    updateDeleteProtection.mutate({
      apiId: api.id,
      enabled: !api.deleteProtection,
    });
  }

  if (api.deleteProtection) {
    return (
      <>
        <Card className="relative">
          <CardHeader>
            <CardTitle>Delete Protection</CardTitle>
            <CardDescription>
              Disabling delete protection will allow you or someone else on the team to delete this
              API, along with all of its keys and data.
            </CardDescription>
          </CardHeader>

          <CardFooter className="z-10 justify-end">
            <Button type="button" onClick={() => setOpen(!open)} variant="destructive">
              Disable Delete Protection
            </Button>
          </CardFooter>
        </Card>
        <Dialog open={open} onOpenChange={(o) => setOpen(o)}>
          <DialogContent className="border-alert">
            <DialogHeader>
              <DialogTitle>Disable Delete Protection</DialogTitle>
              <DialogDescription>
                Disabling delete protection will allow you or someone else on the team to delete
                this API, along with all of its keys and data.
              </DialogDescription>
            </DialogHeader>
            <Form {...form}>
              <form className="flex flex-col space-y-8" onSubmit={form.handleSubmit(onSubmit)}>
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="font-normal text-content-subtle">
                        {" "}
                        Enter the API name{" "}
                        <span className="font-medium text-content">{api.name}</span> to continue:
                      </FormLabel>
                      <FormControl>
                        <Input {...field} autoComplete="off" />
                      </FormControl>

                      <FormMessage />
                    </FormItem>
                  )}
                />

                <DialogFooter>
                  <Button
                    type="button"
                    disabled={updateDeleteProtection.isLoading}
                    onClick={() => {
                      form.reset();
                      setOpen(!open);
                    }}
                  >
                    Cancel
                  </Button>
                  <Button
                    type="submit"
                    variant="destructive"
                    disabled={!isValid || updateDeleteProtection.isLoading}
                  >
                    {updateDeleteProtection.isLoading ? <Loading /> : "Disable"}
                  </Button>
                </DialogFooter>
              </form>
            </Form>
          </DialogContent>
        </Dialog>
      </>
    );
  }

  return (
    <Card className="relative">
      <CardHeader>
        <CardTitle>Delete Protection</CardTitle>
        <CardDescription>
          Delete protection is currently disabeled. You can enable it to prevent this API from
          getting deleted via the dashboard or API.
        </CardDescription>
      </CardHeader>

      <CardFooter className="z-10 justify-end">
        <Button
          onClick={() => updateDeleteProtection.mutate({ apiId: api.id, enabled: true })}
          variant="primary"
          disabled={updateDeleteProtection.isLoading}
          loading={updateDeleteProtection.isLoading}
        >
          Enable
        </Button>
      </CardFooter>
    </Card>
  );
};
