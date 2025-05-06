"use client";
import { ChartPie, CircleInfo } from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { FormInput } from "@unkey/ui";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import type { CreditsFormValues } from "../create-key.schema";
import { ProtectionSwitch } from "./protection-switch";

export const UsageSetup = () => {
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

    // When enabling, ensure refill has the correct structure
    if (checked && !getValues("limit.data.refill.interval")) {
      setValue("limit.data.refill.interval", "none", { shouldValidate: true });
      setValue("limit.data.refill.amount", undefined, { shouldValidate: true });
      setValue("limit.data.refill.refillDay", undefined, {
        shouldValidate: true,
      });
    }

    trigger("limit");
  };

  const handleRefillIntervalChange = (value: "none" | "daily" | "monthly") => {
    setValue("limit.data.refill.interval", value, { shouldValidate: true });

    // Clean up related fields based on the selected interval
    if (value === "none") {
      setValue("limit.data.refill.amount", undefined, { shouldValidate: true });
      setValue("limit.data.refill.refillDay", undefined, {
        shouldValidate: true,
      });
    } else if (value === "daily") {
      // For daily, ensure refillDay is undefined but keep amount if present
      setValue("limit.data.refill.refillDay", undefined, {
        shouldValidate: true,
      });
      // If amount is not set, set a default
      if (!getValues("limit.data.refill.amount")) {
        setValue("limit.data.refill.amount", 100, { shouldValidate: true });
      }
    } else if (value === "monthly") {
      // For monthly, ensure both amount and refillDay have values
      if (!getValues("limit.data.refill.amount")) {
        setValue("limit.data.refill.amount", 100, { shouldValidate: true });
      }
      if (!getValues("limit.data.refill.refillDay")) {
        setValue("limit.data.refill.refillDay", 1, { shouldValidate: true });
      }
    }
  };

  return (
    <div className="space-y-5 px-2 py-1">
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
  );
};
