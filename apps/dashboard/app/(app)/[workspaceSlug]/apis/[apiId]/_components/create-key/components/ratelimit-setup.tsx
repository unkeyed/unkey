"use client";
import { ProtectionSwitch } from "@/components/dashboard/metadata/protection-switch";
import { Gauge, Trash } from "@unkey/icons";
import { Button, FormCheckbox, FormInput, InlineLink } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useEffect } from "react";
import { Controller, useFieldArray, useFormContext, useWatch } from "react-hook-form";
import type { RatelimitFormValues, RatelimitItem } from "../create-key.schema";

export const RatelimitSetup = ({
  overrideEnabled = false,
}: {
  overrideEnabled?: boolean;
}) => {
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
    name: "ratelimit.data" as const,
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
        autoApply: false,
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
      autoApply: false,
    };
    append(newItem);
  };

  return (
    <div className="space-y-5 px-2 py-1">
      {!overrideEnabled && (
        <ProtectionSwitch
          description="Turn on to restrict how frequently this key can be used. Requests
            beyond the limit will be blocked."
          title="Ratelimit"
          icon={<Gauge className="text-gray-12" iconSize="sm-regular" />}
          checked={ratelimitEnabled}
          onCheckedChange={handleSwitchChange}
          {...register("ratelimit.enabled")}
        />
      )}

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

      <div>
        {fields.map((field, index) => (
          <div key={field.id} className="space-y-4 w-full border-t border-grayA-3 py-6">
            <div className="flex items-center gap-[14px] w-full">
              <FormInput
                className={cn(
                  "[&_input:first-of-type]:h-[36px]",
                  fields.length <= 1 ? "w-full" : "flex-1",
                )}
                placeholder="my-ratelimit"
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
                  className="bg-errorA-4 size-[34px] rounded-lg"
                  onClick={() => remove(index)}
                  type="button"
                >
                  <Trash iconSize="sm-regular" className="text-error-11" />
                </Button>
              ) : null}
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <FormInput
                className="hidden"
                type="text"
                hidden
                {...register(`ratelimit.data.${index}.id`)}
              />

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
            </div>

            <Controller
              control={control}
              name={`ratelimit.data.${index}.autoApply`}
              render={({ field }) => (
                <FormCheckbox
                  className={cn(
                    "[&_input:first-of-type]:h-[36px]",
                    fields.length <= 1 ? "w-full" : "flex-1",
                  )}
                  label="Auto Apply"
                  description={
                    <p>
                      This rate limit rule will always be used.{" "}
                      <InlineLink
                        label="Learn more"
                        target="_blank"
                        rel="noopener noreferrer"
                        href="https://unkey.com/docs/apis/features/ratelimiting/overview#auto-apply-vs-manual-ratelimits"
                      />
                      .
                    </p>
                  }
                  error={errors.ratelimit?.data?.[index]?.autoApply?.message}
                  disabled={!ratelimitEnabled}
                  checked={field.value}
                  onCheckedChange={field.onChange}
                  name={field.name}
                  ref={field.ref}
                />
              )}
            />
          </div>
        ))}
      </div>
    </div>
  );
};
