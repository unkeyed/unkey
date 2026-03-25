import { Trash } from "@unkey/icons";
import { Button, FormInput } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { memo } from "react";
import type { UseFormRegister } from "react-hook-form";
import type { EnvVarsFormValues } from "./schema";

type EnvVarRowProps = {
  index: number;
  isOnly: boolean;
  isLast: boolean;
  keyError: string | undefined;
  register: UseFormRegister<EnvVarsFormValues>;
  onRemove: (index: number) => void;
};

export const EnvVarRow = memo(function EnvVarRow({
  index,
  isOnly,
  isLast,
  keyError,
  register,
  onRemove,
}: EnvVarRowProps) {
  return (
    <div className={cn("flex flex-col gap-4")}>
      {/* Key + Value + Delete side by side */}
      <div className="flex items-start gap-4">
        <FormInput
          label="Key"
          className="flex-1 [&_input]:font-mono"
          placeholder="CLIENT_KEY..."
          error={keyError}
          {...register(`envVars.${index}.key`)}
        />
        <FormInput
          label="Value"
          className="flex-1 [&_input]:font-mono"
          placeholder="value"
          {...register(`envVars.${index}.value`)}
        />
        {!isOnly && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="size-9 shrink-0 px-0 justify-center text-error-11 hover:text-error-11 hover:bg-grayA-3 rounded-lg mt-5"
            onClick={() => onRemove(index)}
          >
            <Trash iconSize="sm-regular" />
          </Button>
        )}
      </div>

      {/* Create Note — collapsible */}
      <details className="group">
        <summary className="w-fit text-[13px] text-gray-9 hover:text-gray-12 transition-colors cursor-pointer list-none [&::-webkit-details-marker]:hidden">
          Create Note
        </summary>
        <div className="pt-3">
          <FormInput
            className="[&_input]:text-sm"
            placeholder="Optional description for this variable..."
            {...register(`envVars.${index}.description`)}
          />
        </div>
      </details>
    </div>
  );
});
