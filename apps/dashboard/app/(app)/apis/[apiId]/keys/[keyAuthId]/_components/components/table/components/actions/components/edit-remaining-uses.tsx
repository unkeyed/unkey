import { ProtectionSwitch } from "@/app/(app)/apis/[apiId]/_components/create-key/components/protection-switch";
import {
  type CreditsFormValues,
  creditsSchema,
} from "@/app/(app)/apis/[apiId]/_components/create-key/create-key.schema";
import { getDefaultValues } from "@/app/(app)/apis/[apiId]/_components/create-key/create-key.utils";
import { DialogContainer } from "@/components/dialog-container";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { ChartPie, CircleInfo } from "@unkey/icons";
import { Button, FormInput } from "@unkey/ui";
import { useEffect } from "react";
import { Controller, FormProvider, useWatch } from "react-hook-form";
import { z } from "zod";
import type { ActionComponentProps } from "../keys-table-action.popover";
import { useUpdateKeyRemaining } from "./hooks/use-update-remaining";
import { KeyInfo } from "./key-info";

const editRemainingUsesFormSchema = z.object({
  ...creditsSchema.shape,
});

const EDIT_REMAINING_USES_FORM_STORAGE_KEY = "unkey_edit_remaining_uses_form_state";

type EditRemainingUsesFormValues = CreditsFormValues & {
  originalRemaining?: number;
};
type EditRemainingUsesProps = { keyDetails: KeyDetails } & ActionComponentProps;

export const EditRemainingUses = ({ keyDetails, isOpen, onClose }: EditRemainingUsesProps) => {
  const methods = usePersistedForm<EditRemainingUsesFormValues>(
    EDIT_REMAINING_USES_FORM_STORAGE_KEY,
    {
      resolver: zodResolver(editRemainingUsesFormSchema),
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
      defaultValues: {
        limit: {
          enabled: Boolean(keyDetails.key.remaining) ?? false,
          data: {
            remaining: keyDetails.key.remaining ?? getDefaultValues().limit?.data?.remaining ?? 100,
            refill: keyDetails.key.refillDay
              ? {
                  // Monthly refill
                  interval: "monthly",
                  amount: keyDetails.key.refillAmount ?? 100,
                  refillDay: keyDetails.key.refillDay,
                }
              : keyDetails.key.refillAmount
                ? {
                    // Daily refill
                    interval: "daily",
                    amount: keyDetails.key.refillAmount,
                    refillDay: undefined,
                  }
                : {
                    // No refill
                    interval: "none",
                    amount: undefined,
                    refillDay: undefined,
                  },
          },
        },
      },
    },
    "memory",
  );

  const {
    handleSubmit,
    formState: { isSubmitting, errors, isValid, isDirty },
    register,
    control,
    setValue,
    getValues,
    trigger,
    loadSavedValues,
    saveCurrentValues,
    clearPersistedData,
    reset,
  } = methods;

  // Load saved values when the dialog opens
  useEffect(() => {
    if (isOpen) {
      loadSavedValues();
    }
  }, [isOpen, loadSavedValues]);

  const currentRefillInterval = useWatch({
    control,
    name: "limit.data.refill.interval",
    defaultValue: keyDetails.key.refillDay
      ? "monthly"
      : !keyDetails.key.refillAmount
        ? "daily"
        : "none",
  });

  const handleRefillIntervalChange = (value: "none" | "daily" | "monthly") => {
    if (value === "none") {
      // For "none", set entire refill object
      setValue(
        "limit.data.refill",
        {
          interval: "none",
          amount: undefined,
          refillDay: undefined,
        },
        { shouldValidate: true },
      );
    } else if (value === "daily") {
      // For "daily"
      setValue(
        "limit.data.refill",
        {
          interval: "daily",
          amount: getValues("limit.data.refill.amount") || 100,
          refillDay: undefined, // Must be undefined for daily
        },
        { shouldValidate: true },
      );
    } else if (value === "monthly") {
      // For "monthly"
      setValue(
        "limit.data.refill",
        {
          interval: "monthly",
          amount: getValues("limit.data.refill.amount") || 100,
          refillDay: getValues("limit.data.refill.refillDay") || 1,
        },
        { shouldValidate: true },
      );
    }
  };

  const key = useUpdateKeyRemaining(() => {
    clearPersistedData();
    reset({
      limit: {
        enabled: Boolean(keyDetails.key.remaining) ?? false,
        data: {
          remaining: keyDetails.key.remaining ?? getDefaultValues().limit?.data?.remaining ?? 100,
          refill: keyDetails.key.refillDay
            ? {
                // Monthly refill
                interval: "monthly",
                amount: keyDetails.key.refillAmount ?? 100,
                refillDay: keyDetails.key.refillDay,
              }
            : keyDetails.key.refillAmount
              ? {
                  // Daily refill
                  interval: "daily",
                  amount: keyDetails.key.refillAmount,
                  refillDay: undefined,
                }
              : {
                  // No refill
                  interval: "none",
                  amount: undefined,
                  refillDay: undefined,
                },
        },
      },
    });
    onClose();
  });

  const onSubmit = async (data: EditRemainingUsesFormValues) => {
    try {
      // Map form data to the structure expected by the TRPC endpoint
      await key.mutateAsync({
        keyId: keyDetails.id,
        limitEnabled: Boolean(data.limit?.enabled),
        remaining: data.limit?.enabled ? data.limit.data?.remaining : undefined,
        refill:
          data.limit?.enabled && data.limit.data?.refill.interval !== "none"
            ? {
                amount: data.limit.data?.refill.amount,
                refillDay:
                  data.limit.data?.refill.interval === "monthly"
                    ? data.limit.data?.refill.refillDay
                    : null,
              }
            : undefined,
      });
    } catch {
      // `useEditKeyRemainingUses` already shows a toast, but we still need to
      // prevent unhandled‐rejection noise in the console.
    }
  };

  const limitEnabled = useWatch({
    control,
    name: "limit.enabled",
  });

  const handleSwitchChange = (checked: boolean) => {
    setValue("limit.enabled", checked);

    // When enabling, ensure default values are set properly
    if (checked) {
      // Set default remaining to 100 if not already set
      if (!getValues("limit.data.remaining")) {
        setValue("limit.data.remaining", 100, { shouldValidate: true });
      }

      // Set up refill structure with defaults, using the entire object at once
      if (!getValues("limit.data.refill.interval")) {
        setValue(
          "limit.data.refill",
          {
            interval: "none",
            amount: undefined,
            refillDay: undefined,
          },
          { shouldValidate: true },
        );
      }
    }

    trigger("limit");
  };

  return (
    <FormProvider {...methods}>
      <form id="edit-remaining-uses-form" onSubmit={handleSubmit(onSubmit)}>
        <DialogContainer
          isOpen={isOpen}
          subTitle="Update the number of remaining uses and refill settings for this key"
          onOpenChange={() => {
            saveCurrentValues();
            onClose();
          }}
          title="Edit Remaining Uses"
          footer={
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                form="edit-remaining-uses-form"
                variant="primary"
                size="xlg"
                className="w-full rounded-lg"
                disabled={!isValid || isSubmitting || !isDirty}
              >
                Update remaining uses
              </Button>
              <div className="text-gray-9 text-xs">Changes will be applied immediately</div>
            </div>
          }
        >
          <KeyInfo keyDetails={keyDetails} />
          <div className="py-1 my-2">
            <div className="h-[1px] bg-grayA-3 w-full" />
          </div>
          <div className="space-y-5 py-1">
            <ProtectionSwitch
              description="Turn on to limit how many times this key can be used. Once the limit
            is reached, the key will be disabled."
              title="Credits"
              icon={<ChartPie className="text-gray-12" size="sm-regular" />}
              checked={limitEnabled}
              onCheckedChange={handleSwitchChange}
              {...register("limit.enabled")}
            />
            <FormInput
              className="[&_input:first-of-type]:h-[36px]"
              placeholder="100"
              inputMode="numeric"
              type="number"
              label="Number of uses"
              description="Enter the remaining amount of uses for this key."
              error={errors.limit?.data?.remaining?.message}
              disabled={!limitEnabled}
              readOnly={!limitEnabled}
              {...register("limit.data.remaining")}
            />

            <Controller
              control={control}
              name="limit.data.refill.interval"
              render={({ field }) => (
                <div className="space-y-1.5">
                  <div className="text-gray-11 text-[13px] flex items-center">Refill Rate</div>
                  <Select
                    onValueChange={(value) => {
                      handleRefillIntervalChange(value as "none" | "daily" | "monthly");
                    }}
                    value={field.value || "none"}
                    disabled={!limitEnabled}
                  >
                    <SelectTrigger className="h-9">
                      <SelectValue placeholder="Select refill interval" />
                    </SelectTrigger>
                    <SelectContent className="border-none rounded-md">
                      <SelectItem value="none">None</SelectItem>
                      <SelectItem value="daily">Daily</SelectItem>
                      <SelectItem value="monthly">Monthly</SelectItem>
                    </SelectContent>
                  </Select>
                  <output className="text-gray-9 flex gap-2 items-center text-[13px]">
                    <CircleInfo size="md-regular" aria-hidden="true" />
                    <span>Interval key will be refilled.</span>
                  </output>
                </div>
              )}
            />

            <Controller
              control={control}
              name="limit.data.refill.amount"
              render={({ field }) => (
                <FormInput
                  className="[&_input:first-of-type]:h-[36px]"
                  placeholder="100"
                  inputMode="numeric"
                  type="number"
                  label="Number of uses per interval"
                  description="Enter the number of uses to refill per interval."
                  error={errors.limit?.data?.refill?.amount?.message}
                  disabled={!limitEnabled || currentRefillInterval === "none"}
                  readOnly={!limitEnabled || currentRefillInterval === "none"}
                  value={field.value === undefined ? "" : field.value}
                  onChange={(e) => {
                    const value = e.target.value === "" ? undefined : Number(e.target.value);
                    field.onChange(value);
                  }}
                />
              )}
            />

            <Controller
              control={control}
              name="limit.data.refill.refillDay"
              render={({ field }) => (
                <FormInput
                  className="[&_input:first-of-type]:h-[36px]"
                  placeholder="1"
                  inputMode="numeric"
                  type="number"
                  label="On which day of the month should we refill the key?"
                  description="Enter the day to refill monthly (1-31)."
                  error={errors.limit?.data?.refill?.refillDay?.message}
                  disabled={!limitEnabled || currentRefillInterval !== "monthly"}
                  readOnly={!limitEnabled || currentRefillInterval !== "monthly"}
                  value={field.value === undefined ? "" : field.value}
                  onChange={(e) => {
                    const value = e.target.value === "" ? undefined : Number(e.target.value);
                    field.onChange(value);
                  }}
                />
              )}
            />
          </div>
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
