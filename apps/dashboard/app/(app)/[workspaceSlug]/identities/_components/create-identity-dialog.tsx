"use client";

import { MetadataSetup } from "@/components/dashboard/metadata/metadata-setup";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { metadataSchema } from "@/lib/schemas/metadata";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "@unkey/icons";
import { Button, DialogContainer, FormInput, toast } from "@unkey/ui";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z
  .object({
    externalId: z
      .string()
      .transform((s) => s.trim())
      .refine((trimmed) => trimmed.length >= 3, "External ID must be at least 3 characters")
      .refine((trimmed) => trimmed.length <= 255, "External ID must be 255 characters or fewer")
      .refine((trimmed) => trimmed !== "", "External ID cannot be only whitespace"),
  })
  .merge(metadataSchema);

type FormValues = z.infer<typeof formSchema>;

export function CreateIdentityDialog() {
  const [open, setOpen] = useState(false);
  const utils = trpc.useUtils();

  const methods = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
    defaultValues: {
      externalId: "",
      metadata: {
        enabled: false,
      },
    },
  });

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isValid },
    reset,
  } = methods;

  const createIdentity = trpc.identity.create.useMutation({
    onSuccess: (data) => {
      toast.success("Identity created successfully", {
        description: `Identity "${data.externalId}" has been created.`,
      });
      // Invalidate queries to refetch the list
      utils.identity.query.invalidate();
      setOpen(false);
      reset();
    },
    onError: (error) => {
      if (error.data?.code === "CONFLICT") {
        setError("externalId", {
          message: "An identity with this external ID already exists",
        });
      } else {
        toast.error("Failed to create identity", {
          description: error.message || "An unexpected error occurred",
        });
      }
    },
  });

  const onSubmit = (data: FormValues) => {
    const meta =
      data.metadata?.enabled && data.metadata.data ? JSON.parse(data.metadata.data) : null;
    createIdentity.mutate({
      externalId: data.externalId,
      meta,
    });
  };

  return (
    <>
      <NavbarActionButton title="Create Identity" onClick={() => setOpen(true)}>
        <Plus iconSize="md-medium" />
        Create Identity
      </NavbarActionButton>

      <DialogContainer
        isOpen={open}
        onOpenChange={setOpen}
        title="Create Identity"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="submit"
              form="create-identity-form"
              variant="primary"
              size="xlg"
              disabled={!isValid || createIdentity.isLoading}
              loading={createIdentity.isLoading}
              className="w-full rounded-lg"
            >
              Create Identity
            </Button>
            <div className="text-gray-9 text-xs">
              Create a new identity to associate with keys and rate limits
            </div>
          </div>
        }
      >
        <FormProvider {...methods}>
          <form id="create-identity-form" onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <FormInput
              label="External ID"
              description="A unique identifier for this identity (3-255 characters)"
              error={errors.externalId?.message}
              {...register("externalId")}
              placeholder="user_123 or user@example.com"
              data-1p-ignore
              required
            />

            <MetadataSetup entityType="identity" />
          </form>
        </FormProvider>
      </DialogContainer>
    </>
  );
}
