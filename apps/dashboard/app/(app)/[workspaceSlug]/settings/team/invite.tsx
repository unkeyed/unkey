"use client";

import type { AuthenticatedUser, Organization } from "@/lib/auth/types";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "@unkey/icons";
import {
  Button,
  DialogContainer,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import type React from "react";
import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
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

  const {
    register,
    handleSubmit,
    control,
    getValues,
    formState: { errors, isValid, isSubmitting },
  } = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      role: "basic_member",
    },
  });

  const createInvitation = trpc.org.invitations.create.useMutation({
    onSuccess: () => {
      // Invalidate the invitations list query to trigger a refetch
      utils.org.invitations.list.invalidate();

      toast.success(
        `We have sent an email to ${getValues(
          "email",
        )} with instructions on how to join your workspace.`,
      );
      setDialogOpen(false);
    },
    onError: (error: { message: string }) => {
      toast.error(`Failed to send invitation: ${error.message}`);
    },
  });

  // If user or organization isn't available yet, return null or a loading state
  if (!user.orgId || !organization) {
    return null;
  }

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (!user?.orgId) {
      console.error("Cannot create invitation: user.orgId is missing");
      toast.error("Unable to create invitation. Please refresh and try again.");
      return;
    }

    try {
      await createInvitation.mutateAsync({
        email: values.email,
        role: values.role,
        orgId: user.orgId,
      });
    } catch (error) {
      console.error("Failed to create invitation:", error);
      toast.error("Failed to create invitation. Please try again.");
    }
  }

  return (
    <>
      <Button
        onClick={() => {
          setDialogOpen(!dialogOpen);
        }}
        className="flex-row items-center gap-1 font-semibold h-[38px]"
        {...rest}
        color="default"
      >
        <Plus iconSize="lg-regular" className="w-4 h-4 " />
        Invite Member
      </Button>
      <DialogContainer
        isOpen={dialogOpen}
        onOpenChange={(o) => setDialogOpen(o)}
        title={`Invite someone to join ${organization.name}`}
        subTitle="They will receive an email with instructions on how to join your workspace."
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              form="invite-form"
              variant="primary"
              size="xlg"
              disabled={!isValid || isSubmitting || createInvitation.isLoading}
              loading={isSubmitting || createInvitation.isLoading}
              type="submit"
              className="w-full rounded-lg"
            >
              Send invitation
            </Button>
            <div className="text-gray-9 text-xs">The invitation will be valid for 24 hours</div>
          </div>
        }
      >
        <form id="invite-form" onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-5">
          <FormInput
            label="Email"
            placeholder="hey@unkey.dev"
            error={errors.email?.message}
            {...register("email")}
          />

          <Controller
            control={control}
            name="role"
            render={({ field }) => (
              <div className="space-y-1.5">
                <div className="text-gray-11 text-[13px] flex items-center">Role</div>
                <Select onValueChange={field.onChange} value={field.value}>
                  <SelectTrigger className="h-9">
                    <SelectValue placeholder="Select a role" />
                  </SelectTrigger>
                  <SelectContent className="border-none rounded-md">
                    <SelectItem value="basic_member">Member</SelectItem>
                    <SelectItem value="admin">Admin</SelectItem>
                  </SelectContent>
                </Select>
                {errors.role && (
                  <div className="text-error-11 text-xs ml-0.5">{errors.role.message}</div>
                )}
                <div className="text-gray-9 text-[13px] ml-0.5">
                  Admins may invite new members or remove them and change workspace settings.
                </div>
              </div>
            )}
          />
        </form>
      </DialogContainer>
    </>
  );
};
