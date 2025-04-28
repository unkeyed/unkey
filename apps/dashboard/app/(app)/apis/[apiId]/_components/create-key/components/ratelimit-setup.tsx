"use client";
import { Gauge, Trash } from "@unkey/icons";
import { Button, FormInput } from "@unkey/ui";
import { useEffect } from "react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import type { RatelimitFormValues, RatelimitItem } from "../create-key.schema";
import { ProtectionSwitch } from "./protection-switch";

export const RatelimitSetup = () => {
  const {
    register,
    formState: { errors },
    control,
    setValue,
    trigger,
  } = useFormContext<RatelimitFormValues>();

  // Note: We're using the explicitly defined type from the schema file
  const { fields, append, remove } = useFieldArray({
    control,
    name: "ratelimit.data" as const, // Use as const to make TypeScript recognize this as a literal
  });

  const ratelimitEnabled = useWatch({
    control,
    name: "ratelimit.enabled",
  });

  // Ensure there's always at least one ratelimit item
  useEffect(() => {
    if (fields.length === 0) {
      append({
        name: "Default",
        limit: 10,
        refillInterval: 1000,
      });
    }
  }, [fields.length, append]);

  const handleSwitchChange = (checked: boolean) => {
    setValue("ratelimit.enabled", checked);
    trigger("ratelimit");
  };

  const handleAddRatelimit = () => {
    const newItem: RatelimitItem = {
      name: "",
      limit: 10,
      refillInterval: 1000,
    };
    append(newItem);
  };

  return (
    <div className="space-y-5 px-1 py-1">
      <div className="px-1">
        <ProtectionSwitch
          description="Turn on to restrict how frequently this key can be used. Requests
            beyond the limit will be blocked."
          title="Ratelimit"
          icon={<Gauge className="text-gray-12" size="sm-regular" />}
          checked={ratelimitEnabled}
          onCheckedChange={handleSwitchChange}
          {...register("ratelimit.enabled")}
        />
      </div>

      <div className="flex w-full justify-between items-center px-1">
        <div className="flex gap-2 items-center">
          <span className="font-medium text-sm text-gray-12">Ratelimits</span>
          <span className="rounded-full border border-grayA-3 justify-center items-center flex bg-grayA-3 w-[22px] h-[18px] text-gray-12 text-[11px]">
            {fields.length}
          </span>
        </div>
        <Button
          className="rounded-lg bg-white dark:bg-black text-gray-12 font-medium"
          variant="outline"
          onClick={handleAddRatelimit}
          type="button"
          disabled={!ratelimitEnabled}
        >
          Add additional ratelimit
        </Button>
      </div>

      <div className="max-h-[550px] overflow-y-auto px-1">
        {fields.map((field, index) => (
          <div key={field.id} className="space-y-4 w-full border-t border-grayA-3 py-6">
            <div className="flex items-center gap-[14px] w-full">
              <FormInput
                className="[&_input:first-of-type]:h-[36px] w-full"
                placeholder="Default"
                type="text"
                label="Name"
                description="A name to identify this rate limit rule"
                error={errors.ratelimit?.data?.[index]?.name?.message}
                disabled={!ratelimitEnabled}
                readOnly={!ratelimitEnabled}
                {...register(`ratelimit.data.${index}.name`)}
              />
              {fields.length > 1 ? (
                <Button
                  variant="ghost"
                  color="danger"
                  className="bg-errorA-4"
                  onClick={() => remove(index)}
                  type="button"
                >
                  <Trash size="sm-regular" className="text-error-11" />
                </Button>
              ) : (
                <div className="w-[36px] h-[36px] invisible" />
              )}
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <FormInput
                className="[&_input:first-of-type]:h-[36px]"
                placeholder="10"
                inputMode="numeric"
                type="number"
                label="Limit"
                description="Maximum requests in the given time window"
                error={errors.ratelimit?.data?.[index]?.limit?.message}
                disabled={!ratelimitEnabled}
                readOnly={!ratelimitEnabled}
                {...register(`ratelimit.data.${index}.limit`)}
              />
              <div className="flex items-center gap-4">
                <FormInput
                  className="[&_input:first-of-type]:h-[36px] w-full"
                  label="Refill Interval (ms)"
                  placeholder="1000"
                  inputMode="numeric"
                  type="number"
                  description="Time window in milliseconds"
                  error={errors.ratelimit?.data?.[index]?.refillInterval?.message}
                  disabled={!ratelimitEnabled}
                  readOnly={!ratelimitEnabled}
                  {...register(`ratelimit.data.${index}.refillInterval`)}
                />
                <Button variant="ghost" color="danger" className="bg-errorA-4 invisible">
                  <Trash size="sm-regular" className="text-error-11" />
                </Button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
