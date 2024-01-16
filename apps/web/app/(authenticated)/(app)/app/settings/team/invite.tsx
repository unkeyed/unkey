"use client";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "@/components/ui/toaster";
import { useOrganization } from "@clerk/nextjs";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import React, { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
const formSchema = z.object({
  email: z.string().email(),
  role: z.enum(["admin", "basic_member"]),
});

export const InviteButton = ({ ...rest }: React.ButtonHTMLAttributes<HTMLButtonElement>) => {
  const [dialogOpen, setDialogOpen] = useState(false);

  const { organization } = useOrganization();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      role: "basic_member",
    },
  });

  const [isLoading, setLoading] = useState(false);

  async function onSubmit(values: z.infer<typeof formSchema>) {
    try {
      setLoading(true);
      await organization?.inviteMember({
        emailAddress: values.email,
        role: values.role,
      });

      toast.success(
        `We have sent an email to ${values.email} with instructions on how to join your workspace`,
      );
      router.refresh();
    } catch (err) {
      console.error(err);
      toast.error((err as Error).message);
    } finally {
      setLoading(false);
    }
  }
  const router = useRouter();

  return (
    <>
      <Dialog open={dialogOpen} onOpenChange={(o) => setDialogOpen(o)}>
        <DialogTrigger asChild>
          <Button
            onClick={() => {
              setDialogOpen(!dialogOpen);
            }}
            className="flex-row items-center gap-1 font-semibold "
            {...rest}
          >
            <Plus size={18} className="w-4 h-4 " />
            Invite Member
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Invite someone to join {organization?.name}</DialogTitle>
            <DialogDescription>
              They will receive an email with instructions on how to join your workspace.
            </DialogDescription>
          </DialogHeader>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-4">
              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Email</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="hey@unkey.dev"
                        {...field}
                        className=" dark:focus:border-gray-700"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="role"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Role</FormLabel>
                    <Select onValueChange={field.onChange} defaultValue={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="Select a verified email to display" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="basic_member">Member</SelectItem>
                        <SelectItem value="admin">Admin</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormDescription>
                      Admins may invite new members or remove them and change workspace settings.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <DialogFooter className="flex-row items-center justify-end gap-2 pt-4 ">
                <Button
                  variant="secondary"
                  onClick={() => {
                    setDialogOpen(false);
                  }}
                >
                  Cancel
                </Button>
                <Button
                  variant={form.formState.isValid && !isLoading ? "primary" : "disabled"}
                  disabled={!form.formState.isValid || isLoading}
                  type="submit"
                >
                  {isLoading ? <Loading /> : "Send invitation"}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </>
  );
};
