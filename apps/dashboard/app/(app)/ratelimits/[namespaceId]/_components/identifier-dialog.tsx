"use client";

import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { CircleInfo } from "@unkey/icons";
import { Button, FormInput } from "@unkey/ui";
import type { PropsWithChildren } from "react";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import type { OverrideDetails } from "../types";

const overrideValidationSchema = z.object({
  identifier: z
    .string()
    .trim()
    .min(2, "Name is required and should be at least 2 characters")
    .max(250),
  limit: z.coerce
    .number()
    .int()
    .min(1, "Limit must be at least 1")
    .max(10_000, "Limit cannot exceed 10,000"),
  duration: z.coerce
    .number()
    .int()
    .min(1_000, "Duration must be at least 1 second (1000ms)")
    .max(24 * 60 * 60 * 1000, "Duration cannot exceed 24 hours"),
  async: z.enum(["unset", "sync", "async"]),
});

type FormValues = z.infer<typeof overrideValidationSchema>;

type Props = PropsWithChildren<{
  isModalOpen: boolean;
  onOpenChange: (value: boolean) => void;
  identifier: string;
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
  const { ratelimit } = trpc.useUtils();
  const {
    register,
    handleSubmit,
    control,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(overrideValidationSchema),
    defaultValues: {
      identifier,
      limit: overrideDetails?.limit ?? 10,
      duration: overrideDetails?.duration ?? 60_000,
      async:
        overrideDetails?.async === undefined ? "unset" : overrideDetails.async ? "async" : "sync",
    },
  });

  const update = trpc.ratelimit.override.update.useMutation({
    onSuccess() {
      toast.success("Limits have been updated", {
        description: "Changes may take up to 60s to propagate globally",
      });
      onOpenChange(false);
      ratelimit.overview.logs.query.invalidate();
    },
    onError(err) {
      toast.error("Failed to update override", {
        description: err.message,
      });
    },
  });

  const create = trpc.ratelimit.override.create.useMutation({
    onSuccess() {
      toast.success("Override has been created", {
        description: "Changes may take up to 60s to propagate globally",
      });
      onOpenChange(false);
      ratelimit.overview.logs.query.invalidate();
    },
    onError(err) {
      toast.error("Failed to create override", {
        description: err.message,
      });
    },
  });

  const onSubmitForm = async (values: FormValues) => {
    try {
      const asyncValue = {
        unset: undefined,
        sync: false,
        async: true,
      }[values.async];

      if (overrideDetails?.overrideId) {
        await update.mutateAsync({
          id: overrideDetails.overrideId,
          limit: values.limit,
          duration: values.duration,
          async: Boolean(overrideDetails.async),
        });
      } else {
        await create.mutateAsync({
          namespaceId,
          identifier: values.identifier,
          limit: values.limit,
          duration: values.duration,
          async: asyncValue,
        });
      }
    } catch (error) {
      console.error("Form submission error:", error);
    }
  };

  return (
    <Dialog open={isModalOpen} onOpenChange={onOpenChange}>
      <DialogContent
        className="bg-gray-1 dark:bg-black drop-shadow-2xl border-gray-4 rounded-lg p-0 gap-0"
        onOpenAutoFocus={(e) => {
          // Prevent auto-focus behavior
          e.preventDefault();
        }}
      >
        <DialogHeader className="border-b border-gray-4">
          <DialogTitle className="px-6 py-4 text-gray-12 font-medium text-base">
            Override Identifier
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmitForm)}>
          <div className="flex flex-col gap-4 p-5 pt-4 bg-accent-2">
            <FormInput
              label="Identifier"
              description="The identifier you use when ratelimiting."
              error={errors.identifier?.message}
              {...register("identifier")}
              readOnly
              disabled
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

            <Controller
              control={control}
              name="async"
              render={({ field }) => (
                <div className="space-y-1">
                  <div className="text-gray-11 text-[13px] flex items-center">Override Type</div>

                  <Select onValueChange={field.onChange} value={field.value}>
                    <SelectTrigger className="flex h-8 w-full items-center justify-between rounded-md bg-transparent px-3 py-2 text-[13px] border border-gray-4 focus:border focus:border-gray-4 hover:bg-gray-4 hover:border-gray-8 focus:bg-gray-4">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="border-none">
                      <SelectItem value="unset">Don't override</SelectItem>
                      <SelectItem value="async">Async</SelectItem>
                      <SelectItem value="sync">Sync</SelectItem>
                    </SelectContent>
                  </Select>

                  <output className="text-gray-9 flex gap-2 items-center">
                    <CircleInfo size="md-regular" aria-hidden="true" />
                    <span>Override the mode, async is faster but slightly less accurate.</span>
                  </output>
                </div>
              )}
            />
          </div>

          <DialogFooter className="p-6 border-t border-gray-4">
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                variant="primary"
                size="xlg"
                disabled={isLoading || isSubmitting}
                loading={isLoading || isSubmitting}
                className="w-full rounded-lg"
              >
                Override Identifier
              </Button>
              <div className="text-gray-9 text-xs">
                Changes are propagated globally within 60 seconds
              </div>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
