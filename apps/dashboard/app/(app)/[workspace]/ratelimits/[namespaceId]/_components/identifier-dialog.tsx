"use client";

import { collection } from "@/lib/collections";
import { useWorkspace } from "@/providers/workspace-provider";
import { zodResolver } from "@hookform/resolvers/zod";
import { DuplicateKeyError } from "@tanstack/react-db";
import { Badge, Button, DialogContainer, FormInput } from "@unkey/ui";
import { useRouter } from "next/navigation";
import type { PropsWithChildren } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import type { OverrideDetails } from "../types";

const overrideValidationSchema = z.object({
  identifier: z
    .string()
    .trim()
    .min(2, "Name is required and should be at least 2 characters")
    .max(250),
  limit: z.coerce.number().int().nonnegative().max(10_000, "Limit cannot exceed 10,000"),
  duration: z.coerce
    .number()
    .int()
    .min(1_000, "Duration must be at least 1 second (1000ms)")
    .max(24 * 60 * 60 * 1000, "Duration cannot exceed 24 hours"),
});

type FormValues = z.infer<typeof overrideValidationSchema>;

type Props = PropsWithChildren<{
  isModalOpen: boolean;
  onOpenChange: (value: boolean) => void;
  identifier?: string;
  isLoading?: boolean;
  namespaceId: string;
  overrideDetails?: OverrideDetails | null;
}>;

export const IdentifierDialog = ({
  isModalOpen,
  onOpenChange,
  namespaceId,
  identifier,
  overrideDetails,
  isLoading = false,
}: Props) => {
  const { workspace } = useWorkspace();
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
  } = useForm<FormValues>({
    resolver: zodResolver(overrideValidationSchema),
    defaultValues: {
      identifier,
      limit: overrideDetails?.limit ?? 10,
      duration: overrideDetails?.duration ?? 60_000,
    },
  });

  const router = useRouter();

  const onSubmitForm = async (values: FormValues) => {
    try {
      if (overrideDetails?.overrideId) {
        collection.ratelimitOverrides.update(overrideDetails.overrideId, (draft) => {
          draft.limit = values.limit;
          draft.duration = values.duration;
        });
        onOpenChange(false);
      } else {
        // workaround until tanstack db throws on index violation
        collection.ratelimitOverrides.forEach((override) => {
          if (override.namespaceId === namespaceId && override.identifier === values.identifier) {
            throw new DuplicateKeyError(override.id);
          }
        });
        collection.ratelimitOverrides.insert({
          namespaceId,
          id: new Date().toISOString(), // gets replaced by backend
          identifier: values.identifier,
          limit: values.limit,
          duration: values.duration,
        });
        onOpenChange(false);
        router.push(`/${workspace?.slug}/ratelimits/${namespaceId}/overrides`);
      }
    } catch (error) {
      if (error instanceof DuplicateKeyError) {
        setError("identifier", {
          type: "custom",
          message: "Identifier already exists",
        });
      } else {
        throw error;
      }
    }
  };

  return (
    <DialogContainer
      isOpen={isModalOpen}
      onOpenChange={onOpenChange}
      title="Override Identifier"
      footer={
        <div className="flex flex-col items-center justify-center w-full gap-2">
          <Button
            type="submit"
            form="identifier-form" // Connect to form ID
            variant="primary"
            size="xlg"
            disabled={isLoading || isSubmitting}
            loading={isLoading || isSubmitting}
            className="w-full rounded-lg"
          >
            Override Identifier
          </Button>
          <div className="text-xs text-gray-9">
            Changes are propagated globally within 60 seconds
          </div>
        </div>
      }
    >
      <form
        id="identifier-form"
        onSubmit={handleSubmit(onSubmitForm)}
        className="flex flex-col gap-4"
      >
        <FormInput
          label="Identifier"
          description="The identifier you use when ratelimiting."
          error={errors.identifier?.message}
          {...register("identifier")}
          readOnly={Boolean(identifier)}
          disabled={Boolean(identifier)}
        />

        <FormInput
          label="Limit"
          description="How many requests can be made within a given window."
          error={errors.limit?.message}
          {...register("limit")}
          type="number"
          placeholder="Enter amount (3, 7, 10, 12…)"
        />

        <FormInput
          label="Duration"
          description="Duration of each window in milliseconds."
          error={errors.duration?.message}
          {...register("duration")}
          type="number"
          placeholder="Enter milliseconds (60000, 100000, 1200000…)"
          rightIcon={
            <Badge className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 rounded-md font-mono whitespace-nowrap gap-[6px] font-medium bg-accent-4 text-accent-11 hover:bg-accent-6 ">
              MS
            </Badge>
          }
        />
      </form>
    </DialogContainer>
  );
};
