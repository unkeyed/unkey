"use client";

import { revalidate } from "@/app/actions";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { DialogTrigger } from "@radix-ui/react-dialog";
import { Button } from "@unkey/ui";
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
      toast.success("Permission deleted successfully");
      revalidate("/authorization/permissions");
      router.push("/authorization/permissions");
    },
    onError(err) {
      toast.error(err.message);
    },
  });

  async function onSubmit() {
    deletePermission.mutate({ permissionId: permission.id });
  }

  function handleDialogOpenChange(newState: boolean) {
    setOpen(newState);
    form.reset();
  }

  return (
    <Dialog open={open} onOpenChange={handleDialogOpenChange}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="border-alert p-4 max-w-md mx-auto">
        <DialogHeader>
          <DialogTitle>Delete Permission</DialogTitle>
          <DialogDescription>
            Deleting a permission automatically removes it from all keys.
          </DialogDescription>
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
                    <span className="font-medium text-content break-all">{permission.name}</span> to
                    continue:
                  </FormLabel>
                  <FormControl>
                    <Input {...field} autoComplete="off" className="w-full" />
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
              >
                Cancel
              </Button>
              <Button
                type="submit"
                variant="destructive"
                disabled={!isValid || deletePermission.isLoading}
                loading={deletePermission.isLoading}
              >
                Delete
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};
