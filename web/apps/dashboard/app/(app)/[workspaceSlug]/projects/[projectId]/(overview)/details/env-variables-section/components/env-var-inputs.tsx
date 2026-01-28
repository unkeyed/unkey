import { cn } from "@/lib/utils";
import { Input } from "@unkey/ui";
import { forwardRef } from "react";
import type { FieldErrors, UseFormRegister, UseFormSetValue } from "react-hook-form";
import type { EnvVarFormData } from "../types";

type EnvVarInputsProps = {
  register: UseFormRegister<EnvVarFormData>;
  setValue: UseFormSetValue<EnvVarFormData>;
  errors: FieldErrors<EnvVarFormData>;
  isSecret: boolean;
  keyDisabled?: boolean;
  onKeyDown?: (e: React.KeyboardEvent) => void;
  autoFocus?: boolean;
};

export const EnvVarInputs = forwardRef<HTMLDivElement, EnvVarInputsProps>(
  ({ register, setValue, errors, isSecret, keyDisabled, onKeyDown, autoFocus = false }, ref) => {
    const keyRegister = register("key");

    return (
      <div ref={ref} className="w-fit flex gap-2 items-center">
        <div className="w-[108px]">
          <Input
            {...keyRegister}
            onChange={(e) => {
              if (keyDisabled) {
                return;
              }
              // Auto-uppercase the key and replace spaces with underscores
              // nothing else should be valid in an env var...
              setValue("key", e.target.value.toUpperCase().replace(/ /g, "_"), {
                shouldValidate: true,
              });
            }}
            onKeyDown={onKeyDown}
            placeholder="KEY_NAME"
            disabled={keyDisabled}
            className={cn(
              "min-h-[32px] text-xs w-[108px] font-mono uppercase",
              errors.key && "border-red-6 focus:border-red-7",
              keyDisabled && "opacity-50 cursor-not-allowed",
            )}
            autoFocus={autoFocus}
            autoComplete="off"
            spellCheck={false}
            aria-invalid={Boolean(errors.key)}
          />
        </div>
        <span className="text-gray-9 text-xs px-1">=</span>
        <Input
          {...register("value")}
          onKeyDown={onKeyDown}
          placeholder="Variable value"
          className={cn(
            "min-h-[32px] text-xs flex-1 font-mono",
            errors.value && "border-red-6 focus:border-red-7",
          )}
          type={isSecret ? "password" : "text"}
          autoComplete={isSecret ? "new-password" : "off"}
          spellCheck={false}
          autoCapitalize="none"
          aria-invalid={Boolean(errors.value)}
        />
      </div>
    );
  },
);
