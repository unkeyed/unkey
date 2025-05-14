"use client";
import { DialogContainer } from "@/components/dialog-container";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { toast } from "@/components/ui/toaster";
import type { AuthenticatedUser, Organization } from "@/lib/auth/types";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { FormInput, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { Button } from "@unkey/ui";
import {  Plus } from "lucide-react";
import type React from "react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  email: z.string().email(),
  role: z.enum(["admin", "basic_member"]),
});

interface InviteButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  user: AuthenticatedUser;
  organization: Organization;
}

export const InviteButton = ({ user, organization, ...rest }: InviteButtonProps) => {
  const [dialogOpen, setDialogOpen] = useState(false);
  const utils = trpc.useUtils();

  const createInvitation = trpc.org.invitations.create.useMutation({
    onSuccess: () => {
      // Invalidate the invitations list query to trigger a refetch
      utils.org.invitations.list.invalidate();

      toast.success(
        `We have sent an email to ${form.getValues("email")} with instructions on how to join your workspace.`,
      );
      setDialogOpen(false);
    },
    onError: (error: { message: any }) => {
      toast.error(`Failed to send invitation: ${error.message}`);
    },
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      role: "basic_member",
    },
  });

  // If user or organization isn't available yet, return null or a loading state
  if (!user!.orgId || !organization) {
    return null;
  }

  async function onSubmit(values: z.infer<typeof formSchema>) {
    await createInvitation.mutateAsync({
      email: values.email,
      role: values.role,
      orgId: user!.orgId!,
    });
  }

  return (
    <>
      <Button
        onClick={() => {
          setDialogOpen(!dialogOpen);
        }}
        className="flex-row items-center gap-1 font-semibold "
        {...rest}
        color="default"
      >
        <Plus size={18} className="w-4 h-4 " />
        Invite Member
      </Button>
      <DialogContainer
        isOpen={dialogOpen}
        onOpenChange={(o) => setDialogOpen(o)}
        title={`Invite someone to join ${organization.name}`}
        footer={
          <>
            <Button
              onClick={() => {
                setDialogOpen(false);
              }}
            >
              Cancel
            </Button>
            <Button
              form="invite-form"
              variant="primary"
              disabled={!form.formState.isValid || form.formState.isSubmitting}
              loading={form.formState.isSubmitting}
              type="submit"
            >
              Send invitation
            </Button>
          </>
        }
      >
        <p className="text-sm text-gray-11">
          They will receive an email with instructions on how to join your workspace.
        </p>
        <Form {...form}>
          <form
            id="invite-form"
            onSubmit={form.handleSubmit(onSubmit)}
            className="flex flex-col gap-4"
          >
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Email</FormLabel>
                  <FormControl>
                    <FormInput
                      placeholder="hey@unkey.dev"
                      {...field}
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
          </form>
        </Form>
      </DialogContainer>
    </>
  );
};
