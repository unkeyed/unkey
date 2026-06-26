"use client";
import { ProtectionSwitch } from "@/components/dashboard/metadata/protection-switch";
import { parseDuration } from "@/lib/duration";
import { formatMs } from "@/lib/ms";
import type { RatelimitItem } from "@/lib/schemas/ratelimit";
import { Gauge, Trash } from "@unkey/icons";
import { Button, FormCheckbox, FormInput, InlineLink } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useEffect, useState } from "react";
import { Controller, useFieldArray, useFormContext, useWatch } from "react-hook-form";

// The form's `ratelimit` field is a Zod discriminated union (enabled false/true),
// which makes react-hook-form collapse the `ratelimit.data` field-array element
// type and reject `RatelimitItem`. Typing the form context against the concrete
// runtime shape lets the field-array helpers infer `RatelimitItem` directly,
// avoiding `any` casts. The disabled branch still carries `data` at runtime
// (set via the schema's prefault), so `boolean` is accurate here.
type RatelimitFieldValues = {
  ratelimit: {
    enabled: boolean;
    data: RatelimitItem[];
  };
};

function RefillIntervalField({
  value,
  onChange,
  error,
  disabled,
}: {
  value: number;
  onChange: (ms: number) => void;
  error: string | undefined;
  disabled: boolean;
}) {
  const [display, setDisplay] = useState(() => formatMs(value));
  const [parseError, setParseError] = useState<string>();

  return (
    <FormInput
      className="[&_input:first-of-type]:h-[36px] w-full"
      label="Refill Interval"
      placeholder="e.g. 5s, 2m, 1h, 500ms"
      type="text"
      value={display}
      description={
        value > 0
          ? `Resets every ${formatMs(value, { long: true })}.`
          : "How long before the counter resets."
      }
      error={parseError ?? error}
      disabled={disabled}
      readOnly={disabled}
      onChange={(e) => {
        const raw = e.target.value;
        setDisplay(raw);

        const trimmed = raw.trim();
        if (trimmed === "") {
          setParseError(undefined);
          onChange(0);
          return;
        }

        const asNumber = Number(trimmed);
        if (Number.isFinite(asNumber) && asNumber > 0) {
          setParseError(undefined);
          onChange(Math.floor(asNumber));
          return;
        }

        const parsed = parseDuration(trimmed);
        if (parsed > 0) {
          setParseError(undefined);
          onChange(parsed);
        } else {
          setParseError('Use a duration like "5s", "2m", "1h" or milliseconds');
        }
      }}
    />
  );
}

export const RatelimitSetup = ({
  overrideEnabled = false,
  entityType = "key",
}: {
  overrideEnabled?: boolean;
  entityType?: "key" | "identity";
}) => {
  const {
    register,
    formState: { errors },
    control,
    setValue,
    trigger,
    clearErrors,
  } = useFormContext<RatelimitFieldValues>();

  // Helper to safely access error messages from conditional schema
  const getFieldError = (index: number, field: keyof RatelimitItem): string | undefined => {
    const data = errors.ratelimit?.data;
    if (!data || !Array.isArray(data)) {
      return undefined;
    }
    const fieldError = data[index];
    if (!fieldError || typeof fieldError !== "object") {
      return undefined;
    }
    const error = fieldError[field];
    if (!error || typeof error !== "object") {
      return undefined;
    }
    return "message" in error ? String(error.message) : undefined;
  };

  const { fields, append, remove } = useFieldArray({
    control,
    name: "ratelimit.data",
  });

  const ratelimitEnabled = useWatch({
    control,
    name: "ratelimit.enabled",
  });

  // Ensure there's always at least one ratelimit item
  useEffect(() => {
    if (fields.length === 0) {
      append({
        id: undefined,
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

  const handleAddRatelimit = async () => {
    const newIndex = fields.length;
    append({
      id: undefined,
      name: "",
      limit: 10,
      refillInterval: 1000,
      autoApply: false,
    });
    // Re-run validation so the form/step validity reflects the newly added
    // (incomplete) rule and the submit button disables. Then clear the new
    // rule's errors so we don't flag fields the user hasn't filled in yet.
    // clearErrors only mutates the errors object, leaving isValid untouched.
    await trigger("ratelimit");
    clearErrors(`ratelimit.data.${newIndex}`);
  };

  const description =
    entityType === "key"
      ? "Turn on to restrict how frequently this key can be used. Requests beyond the limit will be blocked."
      : "Turn on to restrict how frequently this identity can be used. Requests beyond the limit will be blocked.";

  return (
    <div className="flex flex-col gap-5 px-2 py-1">
      {!overrideEnabled && (
        <ProtectionSwitch
          description={description}
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
          <div key={field.id} className="flex flex-col gap-4 w-full border-t border-grayA-3 py-6">
            <div className="flex items-center gap-3.5 w-full">
              <FormInput
                className={cn(
                  "[&_input:first-of-type]:h-[36px]",
                  fields.length <= 1 ? "w-full" : "flex-1",
                )}
                placeholder="my-ratelimit"
                type="text"
                label="Name"
                description="A name to identify this rate limit rule"
                error={getFieldError(index, "name")}
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
                error={getFieldError(index, "limit")}
                disabled={!ratelimitEnabled}
                readOnly={!ratelimitEnabled}
                {...register(`ratelimit.data.${index}.limit`)}
              />

              <Controller
                control={control}
                name={`ratelimit.data.${index}.refillInterval`}
                render={({ field }) => (
                  <RefillIntervalField
                    value={field.value}
                    onChange={field.onChange}
                    error={getFieldError(index, "refillInterval")}
                    disabled={!ratelimitEnabled}
                  />
                )}
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
                        href="https://unkey.com/docs/platform/apis/features/ratelimiting/overview#auto-apply-vs-manual-ratelimits"
                      />
                      .
                    </p>
                  }
                  error={getFieldError(index, "autoApply")}
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
