"use client";

import { useProjectScope } from "@/hooks/use-project-scope";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { zodResolver } from "@hookform/resolvers/zod";
import { and, createLiveQueryCollection, eq } from "@tanstack/react-db";
import {
  Badge,
  Button,
  DialogContainer,
  FormField,
  FormInput,
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@unkey/ui";
import { useRouter } from "next/navigation";
import type { PropsWithChildren } from "react";
import type { Resolver } from "react-hook-form";
import { useForm } from "react-hook-form";
import { z } from "zod";
import type { OverrideDetails } from "../types";
import { useOverride } from "./use-override";

const overrideValidationSchema = z.object({
  identifier: z
    .string()
    .trim()
    .min(2, "Name is required and should be at least 2 characters")
    // 512 matches the ratelimit_overrides.identifier column and the API.
    .max(512),
  limit: z.coerce.number().int().nonnegative().max(10_000, "Limit cannot exceed 10,000"),
  duration: z.coerce
    .number()
    .int()
    .min(1_000, "Duration must be at least 1 second (1000ms)")
    .max(30 * 24 * 60 * 60 * 1000, "Duration cannot exceed 30 days"),
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
  const workspace = useWorkspaceNavigation();
  const scope = useProjectScope();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
  } = useForm<FormValues>({
    resolver: zodResolver(overrideValidationSchema) as Resolver<FormValues>,
    defaultValues: {
      identifier,
      limit: overrideDetails?.limit ?? 10,
      duration: overrideDetails?.duration ?? 60_000,
    },
  });

  const router = useRouter();

  const existing = useOverride(namespaceId, identifier);

  const onSubmitForm = async (values: FormValues) => {
    if (overrideDetails?.overrideId) {
      await existing.collection?.toArrayWhenReady();
      collection.ratelimitOverrides.update(overrideDetails.overrideId, (draft) => {
        draft.limit = values.limit;
        draft.duration = values.duration;
      });
      onOpenChange(false);
      return;
    }

    const lookup = createLiveQueryCollection((q) =>
      q
        .from({ override: collection.ratelimitOverrides })
        .where(({ override }) =>
          and(eq(override.namespaceId, namespaceId), eq(override.identifier, values.identifier)),
        ),
    );
    const taken = (await lookup.toArrayWhenReady()).length > 0;
    await lookup.cleanup();
    if (taken) {
      setError("identifier", {
        type: "custom",
        message: "Identifier already exists",
      });
      return;
    }
    collection.ratelimitOverrides.insert({
      namespaceId,
      id: new Date().toISOString(), // gets replaced by backend
      identifier: values.identifier,
      limit: values.limit,
      duration: values.duration,
    });
    onOpenChange(false);
    router.push(
      routes.ratelimits.overrides({ workspaceSlug: workspace.slug, ...scope, namespaceId }),
    );
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
          className="secret"
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

        <FormField
          label="Duration"
          description="Duration of each window in milliseconds."
          error={errors.duration?.message}
        >
          {(field) => (
            <InputGroup variant={field.variant}>
              <InputGroupInput
                id={field.id}
                aria-describedby={field.describedBy}
                aria-invalid={field.invalid}
                {...register("duration")}
                type="number"
                placeholder="Enter milliseconds (60000, 100000, 1200000…)"
              />
              <InputGroupAddon align="inline-end">
                <Badge className="pointer-events-none rounded-md font-mono whitespace-nowrap gap-[6px] font-medium bg-gray-4 text-gray-11 hover:bg-gray-6">
                  MS
                </Badge>
              </InputGroupAddon>
            </InputGroup>
          )}
        </FormField>
      </form>
    </DialogContainer>
  );
};
