"use client";
import { ProtectionSwitch } from "@/components/dashboard/metadata/protection-switch";
import { ChartPie } from "@unkey/icons";
import {
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { FormDescription } from "@unkey/ui/src/components/form/form-helpers";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import type { CreditsFormValues } from "../create-key.schema";

export const UsageSetup = ({
  overrideEnabled = false,
}: {
  overrideEnabled?: boolean;
}) => {
  const {
    register,
    formState: { errors },
    control,
    setValue,
    getValues,
    trigger,
  } = useFormContext<CreditsFormValues>();

  const limitEnabled = useWatch({
    control,
    name: "limit.enabled",
  });

  const currentRefillInterval = useWatch({
    control,
    name: "limit.data.refill.interval",
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

  return (
    <div className="space-y-5 px-2 py-1">
      {!overrideEnabled && (
        <ProtectionSwitch
          description="Turn on to limit how many times this key can be used. Once the limit
            is reached, the key will be disabled."
          title="Credits"
          icon={<ChartPie className="text-gray-12" iconSize="sm-regular" />}
          checked={limitEnabled}
          onCheckedChange={handleSwitchChange}
          {...register("limit.enabled")}
        />
      )}

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
            <FormDescription
              description="Interval key will be refilled."
              descriptionId="refill-interval-description"
              errorId="refill-interval-error"
              error={errors.limit?.data?.refill?.interval?.message}
            />
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
  );
};
