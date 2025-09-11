import { cn } from "@/lib/utils";
import { Input } from "@unkey/ui";
import { forwardRef } from "react";
import type { UseFormRegister, FieldErrors } from "react-hook-form";
import type { EnvVarFormData } from "../types";

type EnvVarInputsProps = {
  register: UseFormRegister<EnvVarFormData>;
  errors: FieldErrors<EnvVarFormData>;
  isSecret: boolean;
  onKeyDown?: (e: React.KeyboardEvent) => void;
  autoFocus?: boolean;
};

export const EnvVarInputs = forwardRef<HTMLDivElement, EnvVarInputsProps>(
  ({ register, errors, isSecret, onKeyDown, autoFocus = false }, ref) => {
    return (
      <div ref={ref} className="w-fit flex gap-2 items-center">
        <div className="w-[108px]">
          <Input
            {...register("key")}
            onKeyDown={onKeyDown}
            placeholder="Variable name"
            className={cn(
              "min-h-[32px] text-xs w-[108px] font-mono",
              errors.key && "border-red-6 focus:border-red-7"
            )}
            autoFocus={autoFocus}
          />
        </div>
        <span className="text-gray-9 text-xs px-1">=</span>
        <Input
          {...register("value")}
          onKeyDown={onKeyDown}
          placeholder="Variable value"
          className={cn(
            "min-h-[32px] text-xs flex-1 font-mono",
            errors.value && "border-red-6 focus:border-red-7"
          )}
          type={isSecret ? "password" : "text"}
        />
      </div>
    );
  }
);
