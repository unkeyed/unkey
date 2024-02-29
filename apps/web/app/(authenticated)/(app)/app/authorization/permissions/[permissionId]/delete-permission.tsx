"use client";

import { Loading } from "@/components/dashboard/loading";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { DialogTrigger } from "@radix-ui/react-dialog";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

type Props = {
  trigger: React.ReactNode;
  permission: {
    id: string;
    name: string;
  };
};

export const DeletePermission: React.FC<Props> = ({ trigger, permission }) => {
  const router = useRouter();

  const [open, setOpen] = useState(false);

  const formSchema = z.object({
    name: z.string().refine((v) => v === permission.name, "Please confirm the role's name"),
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const isValid = form.watch("name") === permission.name;

  const deletePermission = trpc.rbac.deletePermission.useMutation({
    onSuccess() {
      toast.success("Role deleted");
      router.push("/app/authorization/permissions");
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit() {
    deletePermission.mutate({ permissionId: permission.id });
  }

  return (
    <Dialog open={open} onOpenChange={(o) => setOpen(o)}>
      <DialogTrigger>{trigger}</DialogTrigger>
      <DialogContent className="border-alert">
        <DialogHeader>
          <DialogTitle>Delete Permission</DialogTitle>
          <DialogDescription>XXX</DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form className="flex flex-col space-y-8" onSubmit={form.handleSubmit(onSubmit)}>
            <Alert variant="alert">
              <AlertTitle>Warning</AlertTitle>
              <AlertDescription>This action is not reversible. Please be certain.</AlertDescription>
            </Alert>

            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="font-normal text-content-subtle">
                    {" "}
                    Enter the permission's name{" "}
                    <span className="font-medium text-content">{permission.name}</span> to continue:
                  </FormLabel>
                  <FormControl>
                    <Input {...field} autoComplete="off" />
                  </FormControl>

                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter className="justify-end gap-4">
              <Button
                type="button"
                disabled={deletePermission.isLoading}
                onClick={() => setOpen(!open)}
                variant="secondary"
              >
                Cancel
              </Button>
              <Button
                type="submit"
                variant={isValid ? "alert" : "disabled"}
                disabled={!isValid || deletePermission.isLoading}
              >
                {deletePermission.isLoading ? <Loading /> : "Delete"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};
