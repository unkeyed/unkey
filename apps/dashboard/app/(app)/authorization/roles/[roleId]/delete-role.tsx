"use client";

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
  role: {
    id: string;
    name: string;
  };
};

export const DeleteRole: React.FC<Props> = ({ trigger, role }) => {
  const router = useRouter();

  const [open, setOpen] = useState(false);

  const formSchema = z.object({
    name: z.string().refine((v) => v === role.name, "Please confirm the role's name"),
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const isValid = form.watch("name") === role.name;

  const deleteRole = trpc.rbac.deleteRole.useMutation({
    onMutate() {
      toast.loading("Deleting Role");
    },
    onSuccess() {
      toast.success("Role deleted successfully");
      router.push("/authorization/roles");
    },
    onError(err) {
      toast.error(err.message);
    },
  });

  async function onSubmit() {
    deleteRole.mutate({ roleId: role.id });
  }

  function handleDialogOpenChange(newState: boolean) {
    setOpen(newState);
    form.reset();
  }

  return (
    <Dialog open={open} onOpenChange={handleDialogOpenChange}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="border-alert">
        <DialogHeader>
          <DialogTitle>Delete Role</DialogTitle>
          <DialogDescription>
            This role will be deleted, keys with this role will be disconnected from all permissions
            granted by this role.
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
                    Enter the role's key{" "}
                    <span className="font-medium text-content">{role.name}</span> to continue:
                  </FormLabel>
                  <FormControl>
                    <Input {...field} autoComplete="off" />
                  </FormControl>

                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter className="justify-end gap-4">
              <Button type="button" disabled={deleteRole.isLoading} onClick={() => setOpen(!open)}>
                Cancel
              </Button>
              <Button
                type="submit"
                variant="destructive"
                disabled={!isValid || deleteRole.isLoading}
                loading={deleteRole.isLoading}
              >
                Delete Role
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};
