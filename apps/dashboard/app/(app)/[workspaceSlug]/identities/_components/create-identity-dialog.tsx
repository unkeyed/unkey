"use client";

import { NavbarActionButton } from "@/components/navigation/action-button";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "@unkey/icons";
import { Button, DialogContainer, FormInput, toast } from "@unkey/ui";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  externalId: z
    .string()
    .min(3, "External ID must be at least 3 characters")
    .max(255, "External ID must be less than 255 characters")
    .trim()
    .refine((id) => !/^\s+$/.test(id), "External ID cannot be only whitespace"),
  meta: z
    .string()
    .optional()
    .refine(
      (val) => {
        if (!val || val.trim() === "") {
          return true;
        }
        try {
          JSON.parse(val);
          // Check size limit (1MB)
          const size = new Blob([val]).size;
          return size < 1024 * 1024;
        } catch {
          return false;
        }
      },
      {
        message: "Must be valid JSON and less than 1MB",
      },
    ),
});

type FormValues = z.infer<typeof formSchema>;

export function CreateIdentityDialog() {
  const [open, setOpen] = useState(false);
  const utils = trpc.useUtils();

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isValid },
    reset,
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
    defaultValues: {
      externalId: "",
      meta: "",
    },
  });

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
    const meta = data.meta?.trim() ? JSON.parse(data.meta) : null;
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

          <div className="space-y-2">
            <label className="text-sm font-medium text-accent-12" htmlFor="meta">
              Metadata (Optional)
            </label>
            <textarea
              id="meta"
              className="w-full min-h-[120px] px-3 py-2 text-xs font-mono rounded-md border border-gray-6 bg-background focus:outline-none focus:ring-2 focus:ring-gray-8"
              placeholder='{"plan": "pro", "email": "user@example.com"}'
              {...register("meta")}
              data-1p-ignore
            />
            {errors.meta && <p className="text-xs text-error-11">{errors.meta.message}</p>}
            <p className="text-xs text-gray-9">
              Optional JSON metadata (must be valid JSON, max 1MB)
            </p>
          </div>
        </form>
      </DialogContainer>
    </>
  );
}
