"use client";
import { Gauge } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useFormContext, useWatch } from "react-hook-form";
import type { RatelimitFormValues } from "../schema";
import { ProtectionSwitch } from "./protection-switch";

export const RatelimitSetup = () => {
  const {
    register,
    formState: { errors },
    control,
    setValue,
  } = useFormContext<RatelimitFormValues>();

  const ratelimitEnabled = useWatch({
    control,
    name: "ratelimit.enabled",
    defaultValue: false,
  });
  const handleSwitchChange = (checked: boolean) => {
    setValue("ratelimit.enabled", checked);
  };

  return (
    <div className="space-y-5 px-2 py-1">
      <ProtectionSwitch
        description="Turn on to restrict how frequently this key can be used. Requests
            beyond the limit will be blocked."
        title="Ratelimit"
        icon={<Gauge className="text-gray-12" size="sm-regular" />}
        checked={ratelimitEnabled}
        onCheckedChange={handleSwitchChange}
        {...register("ratelimit.enabled")}
      />
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
