"use client";

import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import { Button } from "@unkey/ui";
import type { PropsWithChildren, ReactNode } from "react";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import type { OverrideDetails } from "../../../logs-table";
import { InputTooltip } from "./input-tooltip";

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

type FormFieldProps = {
  label: string;
  tooltip?: string;
  error?: string;
  children: ReactNode;
};

const FormField = ({ label, tooltip, error, children }: FormFieldProps) => (
  // biome-ignore lint/a11y/useKeyWithClickEvents: no need for button
  <div className="flex flex-col gap-1" onClick={(e) => e.stopPropagation()}>
    <Label
      className="text-gray-11 text-[13px] flex items-center"
      onClick={(e) => e.preventDefault()}
    >
      {label}
      {tooltip && (
        <InputTooltip desc={tooltip}>
          <CircleInfo size="md-regular" className="text-accent-8 ml-[10px]" />
        </InputTooltip>
      )}
    </Label>
    {children}
    {error && (
      <span className="text-error-10 text-[13px] font-medium">{error}</span>
    )}
  </div>
);

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
        overrideDetails?.async === undefined
          ? "unset"
          : overrideDetails.async
          ? "async"
          : "sync",
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
          <div className="flex flex-col gap-4 py-4 px-6 bg-accent-2">
            <FormField
              label="Identifier"
              tooltip="The identifier you use when ratelimiting."
              error={errors.identifier?.message}
            >
              <Input
                {...register("identifier")}
                readOnly
                disabled
                className="border-gray-4 focus:border-gray-5 px-3 py-1"
              />
            </FormField>

            <FormField
              label="Limit"
              tooltip="How many requests can be made within a given window."
              error={errors.limit?.message}
            >
              <Input
                {...register("limit")}
                type="number"
                placeholder="Enter amount (3, 7, 10, 12…)"
                className="border border-gray-4 focus:border focus:border-gray-4 px-3 py-1 hover:bg-gray-4 hover:border-gray-8 focus:bg-gray-4 rounded-md"
              />
            </FormField>

            <FormField
              label="Duration"
              tooltip="Duration of each window in milliseconds."
              error={errors.duration?.message}
            >
              <div className="relative">
                <Input
                  {...register("duration")}
                  type="number"
                  placeholder="Enter milliseconds (60000, 100000, 1200000…)"
                  className="border border-gray-4 focus:border focus:border-gray-4 px-3 py-1 hover:bg-gray-4 hover:border-gray-8 focus:bg-gray-4 rounded-md"
                />
                <Badge className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 rounded-md font-mono whitespace-nowrap gap-[6px] font-medium bg-accent-4 text-accent-11 hover:bg-accent-6 ">
                  MS
                </Badge>
              </div>
            </FormField>

            <FormField
              label="Override Type"
              tooltip="Override the mode, async is faster but slightly less accurate."
              error={errors.async?.message}
            >
              <Controller
                control={control}
                name="async"
                render={({ field }) => (
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
                )}
              />
            </FormField>
          </div>

          <DialogFooter className="px-6 py-4 border-t border-gray-4">
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                variant="primary"
                disabled={isLoading || isSubmitting}
                loading={isLoading || isSubmitting}
                className="h-10 w-full rounded-lg"
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
