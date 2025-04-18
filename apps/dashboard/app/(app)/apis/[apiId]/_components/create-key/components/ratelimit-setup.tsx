"use client";
import { Switch } from "@/components/ui/switch";
import { Gauge } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useEffect, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import type { RatelimitFormValues } from "../schema";

export const RatelimitSetup = () => {
  const {
    register,
    formState: { errors },
    control,
    setValue,
    getValues,
  } = useFormContext<RatelimitFormValues>();

  const [lastLimit, setLastLimit] = useState<number | null>(null);
  const [lastInterval, setLastInterval] = useState<number | null>(null);

  const ratelimitEnabled = useWatch({
    control,
    name: "ratelimit.enabled",
    defaultValue: false,
  });

  const currentLimit = useWatch({
    control,
    name: "ratelimit.data.limit",
  });

  const currentInterval = useWatch({
    control,
    name: "ratelimit.data.refillInterval",
  });

  useEffect(() => {
    if (ratelimitEnabled && currentLimit) {
      setLastLimit(currentLimit);
    }
  }, [ratelimitEnabled, currentLimit]);

  useEffect(() => {
    if (ratelimitEnabled && currentInterval) {
      setLastInterval(currentInterval);
    }
  }, [ratelimitEnabled, currentInterval]);

  const handleSwitchChange = (checked: boolean) => {
    setValue("ratelimit.enabled", checked);
    if (checked) {
      if (lastLimit) {
        setValue("ratelimit.data.limit", lastLimit);
      } else if (!getValues("ratelimit.data.limit")) {
        setValue("ratelimit.data.limit", 10);
      }

      if (lastInterval) {
        setValue("ratelimit.data.refillInterval", lastInterval);
      } else if (!getValues("ratelimit.data.refillInterval")) {
        setValue("ratelimit.data.refillInterval", 1000);
      }
    } else {
      setValue("ratelimit.data", undefined);
    }
  };

  return (
    <div className="space-y-5 px-2 py-1">
      <div className="flex flex-row py-5 pl-5 pr-[26px] gap-14 justify-between border rounded-xl border-grayA-5 bg-white dark:bg-black items-center">
        <div className="flex flex-col gap-4">
          <div className="flex gap-3">
            <div className="p-1.5 bg-grayA-3 rounded-md border border-grayA-3">
              <Gauge className="text-gray-12" size="sm-regular" />
            </div>
            <div className="text-sm font-medium text-gray-12">Ratelimit</div>
          </div>
          <div className="text-gray-9 text-xs">
            Turn on to restrict how frequently this key can be used. Requests beyond the limit will
            be blocked.
          </div>
        </div>
        <Switch
          checked={ratelimitEnabled}
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
          {...register("ratelimit.enabled")}
        />
      </div>
      <FormInput
        className="[&_input:first-of-type]:h-[36px]"
        placeholder="10"
        inputMode="numeric"
        type="number"
        label="Limit"
        description="The maximum number of requests in the given fixed window."
        error={errors.ratelimit?.data?.limit?.message}
        disabled={!ratelimitEnabled}
        readOnly={!ratelimitEnabled}
        {...register("ratelimit.data.limit")}
      />
      <FormInput
        className="[&_input:first-of-type]:h-[36px]"
        label="Refill Interval (milliseconds)"
        placeholder="1000"
        inputMode="numeric"
        type="number"
        description="The time window in milliseconds for the rate limit to reset."
        error={errors.ratelimit?.data?.refillInterval?.message}
        disabled={!ratelimitEnabled}
        readOnly={!ratelimitEnabled}
        {...register("ratelimit.data.refillInterval")}
      />
    </div>
  );
};
