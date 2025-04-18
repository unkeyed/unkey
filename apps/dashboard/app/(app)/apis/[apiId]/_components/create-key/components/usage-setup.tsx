"use client";
import { FormInput } from "@unkey/ui";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import { useState, useEffect } from "react";
import type { LimitFormValues } from "../schema";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ChartPie, CircleInfo } from "@unkey/icons";

export const UsageSetup = () => {
  const {
    register,
    formState: { errors },
    control,
    setValue,
    getValues,
  } = useFormContext<LimitFormValues>();

  // Store the last used values in component state
  const [lastRemaining, setLastRemaining] = useState<number | null>(null);
  const [lastRefillAmount, setLastRefillAmount] = useState<number | null>(null);
  const [lastRefillInterval, setLastRefillInterval] = useState<
    "none" | "daily" | "monthly"
  >("none");
  const [lastRefillDay, setLastRefillDay] = useState<number | null>(null);

  const limitEnabled = useWatch({
    control,
    name: "limitEnabled",
    defaultValue: false,
  });

  // Watch values to store when changed
  const currentRemaining = useWatch({
    control,
    name: "limit.remaining",
  });

  const currentRefillAmount = useWatch({
    control,
    name: "limit.refill.amount",
  });

  const currentRefillInterval = useWatch({
    control,
    name: "limit.refill.interval",
    defaultValue: "none",
  });

  const currentRefillDay = useWatch({
    control,
    name: "limit.refill.refillDay",
  });

  // Store values when they change and limit is enabled
  useEffect(() => {
    if (limitEnabled && currentRemaining) {
      setLastRemaining(currentRemaining);
    }
  }, [limitEnabled, currentRemaining]);

  useEffect(() => {
    if (limitEnabled && currentRefillAmount) {
      setLastRefillAmount(currentRefillAmount);
    }
  }, [limitEnabled, currentRefillAmount]);

  useEffect(() => {
    if (limitEnabled && currentRefillInterval) {
      setLastRefillInterval(currentRefillInterval);
    }
  }, [limitEnabled, currentRefillInterval]);

  useEffect(() => {
    if (limitEnabled && currentRefillDay) {
      setLastRefillDay(currentRefillDay);
    }
  }, [limitEnabled, currentRefillDay]);

  const handleSwitchChange = (checked: boolean) => {
    setValue("limitEnabled", checked);

    if (checked) {
      // When enabling, restore last used values if they exist
      if (lastRemaining) {
        setValue("limit.remaining", lastRemaining);
      } else if (!getValues("limit.remaining")) {
        setValue("limit.remaining", 100);
      }

      if (lastRefillInterval) {
        setValue("limit.refill.interval", lastRefillInterval);
      } else {
        setValue("limit.refill.interval", "none");
      }

      if (lastRefillAmount && lastRefillInterval !== "none") {
        setValue("limit.refill.amount", lastRefillAmount);
      }

      if (lastRefillDay && lastRefillInterval === "monthly") {
        setValue("limit.refill.refillDay", lastRefillDay);
      } else if (
        lastRefillInterval === "monthly" &&
        !getValues("limit.refill.refillDay")
      ) {
        setValue("limit.refill.refillDay", 1);
      }
    } else {
      // When disabling, set limit to undefined but keep our saved values
      setValue("limit", undefined);
    }
  };

  const handleRefillIntervalChange = (value: "none" | "daily" | "monthly") => {
    setValue("limit.refill.interval", value);

    if (value === "none") {
      setValue("limit.refill.amount", undefined);
      setValue("limit.refill.refillDay", undefined);
    } else if (value === "monthly") {
      if (lastRefillDay) {
        setValue("limit.refill.refillDay", lastRefillDay);
      } else if (!getValues("limit.refill.refillDay")) {
        setValue("limit.refill.refillDay", 1);
      }
    } else if (value === "daily") {
      setValue("limit.refill.refillDay", undefined);
    }
  };

  return (
    <div className="space-y-5 px-2 py-1">
      <div className="flex flex-row py-5 pl-5 pr-[26px] gap-14 justify-between border rounded-xl border-grayA-5 bg-white dark:bg-black items-center">
        <div className="flex flex-col gap-4">
          <div className="flex gap-3">
            <div className="p-1.5 bg-grayA-3 rounded-md border border-grayA-3">
              <ChartPie className="text-gray-12" size="sm-regular" />
            </div>
            <div className="text-sm font-medium text-gray-12">Limited Use</div>
          </div>
          <div className="text-gray-9 text-xs">
            Turn on to limit how many times this key can be used. Once the limit
            is reached, the key will be disabled.
          </div>
        </div>
        <Switch
          checked={limitEnabled}
          onCheckedChange={handleSwitchChange}
          className="
            h-4 w-7
            data-[state=checked]:bg-success-9
            data-[state=checked]:ring-2
            data-[state=checked]:ring-successA-5
            data-[state=unchecked]:bg-gray-3
            data-[state=unchecked]:ring-2
            data-[state=unchecked]:ring-grayA-3
            [&>span]:h-3.5 [&>span]:w-3.5
          "
          {...register("limitEnabled")}
        />
      </div>

      <FormInput
        className="[&_input:first-of-type]:h-[36px]"
        placeholder="100"
        inputMode="numeric"
        type="number"
        label="Number of uses"
        description="Enter the remaining amount of uses for this key."
        error={errors.limit?.remaining?.message}
        disabled={!limitEnabled}
        readOnly={!limitEnabled}
        {...register("limit.remaining")}
      />

      <Controller
        control={control}
        name="limit.refill.interval"
        render={({ field }) => (
          <div className="space-y-1.5">
            <div className="text-gray-11 text-[13px] flex items-center">
              Refill Rate
            </div>
            <Select
              onValueChange={(value) => {
                field.onChange(value);
                handleRefillIntervalChange(
                  value as "none" | "daily" | "monthly"
                );
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

      <FormInput
        className="[&_input:first-of-type]:h-[36px]"
        placeholder="100"
        inputMode="numeric"
        type="number"
        label="Number of uses per interval"
        description="Enter the number of uses to refill per interval."
        error={errors.limit?.refill?.amount?.message}
        disabled={!limitEnabled || currentRefillInterval === "none"}
        readOnly={!limitEnabled || currentRefillInterval === "none"}
        {...register("limit.refill.amount")}
      />

      <FormInput
        className="[&_input:first-of-type]:h-[36px]"
        placeholder="1"
        inputMode="numeric"
        type="number"
        label="On which day of the month should we refill the key?"
        description="Enter the day to refill monthly (1-31)."
        error={errors.limit?.refill?.refillDay?.message}
        disabled={!limitEnabled || currentRefillInterval !== "monthly"}
        readOnly={!limitEnabled || currentRefillInterval !== "monthly"}
        {...register("limit.refill.refillDay")}
      />
    </div>
  );
};
