"use client";

import type { Organization } from "@/lib/auth/types";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus, XMark } from "@unkey/icons";
import {
  Button,
  Card,
  CardContent,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { Controller, useFieldArray, useForm } from "react-hook-form";
import { z } from "zod";

const inviteSchema = z.object({
  invites: z.array(
    z.object({
      email: z.string().email("Invalid email address"),
      role: z.enum(["admin", "basic_member"]),
    }),
  ),
});

type InviteFormProps = {
  organization: Organization;
};

export const InviteForm = ({ organization }: InviteFormProps) => {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const utils = trpc.useUtils();

  const {
    register,
    handleSubmit,
    control,
    reset,
    formState: { errors },
  } = useForm<z.infer<typeof inviteSchema>>({
    resolver: zodResolver(inviteSchema),
    defaultValues: {
      invites: [{ email: "", role: "basic_member" as const }],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control,
    name: "invites",
  });

  const createInvitation = trpc.org.invitations.create.useMutation();

  async function onSubmit(values: z.infer<typeof inviteSchema>) {
    setIsSubmitting(true);

    try {
      const results = await Promise.allSettled(
        values.invites.map((invite) =>
          createInvitation.mutateAsync({
            email: invite.email,
            role: invite.role,
            orgId: organization.id,
          }),
        ),
      );

      const successful = results.filter((r) => r.status === "fulfilled").length;
      const failed = results
        .map((result, index) => {
          if (result.status === "rejected") {
            const error = result.reason as {
              message?: string;
              data?: { message?: string; code?: string };
              shape?: { message?: string };
            };

            // Try to extract the most useful error message
            let errorMessage =
              error.data?.message || error.shape?.message || error.message || "Unknown error";

            // If the error is too generic, provide more context
            if (errorMessage === "Failed to invite member") {
              errorMessage =
                "Unable to send invitation. They may already be a member or have a pending invitation.";
            }

            return {
              email: values.invites[index].email,
              error: errorMessage,
            };
          }
          return null;
        })
        .filter((item): item is { email: string; error: string } => item !== null);

      if (successful > 0) {
        utils.org.invitations.list.invalidate();

        if (failed.length > 0) {
          if (failed.length === 1) {
            toast.success(
              `Sent ${successful} invitation${successful > 1 ? "s" : ""}, but failed to invite ${failed[0].email}: ${failed[0].error}`,
            );
          } else {
            const failedEmails = failed.map((f) => f.email).join(", ");
            toast.success(
              `Sent ${successful} invitation${successful > 1 ? "s" : ""}, but failed to invite: ${failedEmails}`,
            );
          }
        } else {
          toast.success(`Successfully sent ${successful} invitation${successful > 1 ? "s" : ""}`);
        }

        reset({ invites: [{ email: "", role: "basic_member" as const }] });
      } else {
        if (failed.length === 1) {
          toast.error(`Failed to invite ${failed[0].email}: ${failed[0].error}`);
        } else {
          const failedList = failed.map((f) => `${f.email} (${f.error})`).join(", ");
          toast.error(`All invitations failed: ${failedList}`);
        }
      }
    } catch (error) {
      console.error("Failed to create invitations:", error);
      toast.error("Failed to create invitations. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <Card>
      <CardContent className="p-6">
        <div className="mb-6">
          <h3 className="text-base font-medium text-content">
            Invite new members by email address
          </h3>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-3">
            {fields.map((field, index) => (
              <div key={field.id} className="flex gap-3 items-end">
                <div className="flex-1">
                  <label
                    htmlFor={`invites.${index}.email`}
                    className="text-sm text-content-subtle mb-1.5 block"
                  >
                    Email
                  </label>
                  <FormInput
                    id={`invites.${index}.email`}
                    placeholder="jane@example.com"
                    className="email"
                    error={errors.invites?.[index]?.email?.message}
                    {...register(`invites.${index}.email`)}
                  />
                </div>
                <div className="w-48">
                  <label
                    htmlFor={`invites.${index}.role`}
                    className="text-sm text-content-subtle mb-1.5 block"
                  >
                    Role
                  </label>
                  <Controller
                    control={control}
                    name={`invites.${index}.role`}
                    render={({ field: roleField }) => (
                      <Select onValueChange={roleField.onChange} value={roleField.value}>
                        <SelectTrigger id={`invites.${index}.role`}>
                          <SelectValue placeholder="Select role" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="basic_member">Member</SelectItem>
                          <SelectItem value="admin">Admin</SelectItem>
                        </SelectContent>
                      </Select>
                    )}
                  />
                </div>
                {fields.length > 1 && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => remove(index)}
                    aria-label={`Remove invite ${index + 1}`}
                  >
                    <XMark className="w-4 h-4" aria-hidden="true" />
                  </Button>
                )}
              </div>
            ))}
          </div>

          <div className="border-t border-border pt-4">
            <div className="flex items-center justify-between">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => append({ email: "", role: "basic_member" as const })}
              >
                <Plus className="w-4 h-4" />
                <span>Add more</span>
              </Button>

              <Button type="submit" disabled={isSubmitting} loading={isSubmitting}>
                Invite
              </Button>
            </div>
          </div>
        </form>
      </CardContent>
    </Card>
  );
};
