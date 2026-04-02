import { Plus, Trash } from "@unkey/icons";
import { Button, FormInput } from "@unkey/ui";
import type { FieldErrors, UseFormRegister } from "react-hook-form";
import type { EnvVarsFormValues } from "./schema";

type EnvVarRowProps = {
  index: number;
  isOnly: boolean;
  register: UseFormRegister<EnvVarsFormValues>;
  onRemove: (index: number) => void;
  errors?: FieldErrors<EnvVarsFormValues>["envVars"];
};

export const EnvVarRow = ({ index, isOnly, register, onRemove, errors }: EnvVarRowProps) => {
  const fieldErrors = errors?.[index];

  return (
    <div className="flex flex-col gap-3">
      {/* Key + Value + Delete side by side */}
      <div className="flex items-start gap-4">
        <FormInput
          label="Key"
          className="flex-1 [&_input]:font-mono"
          placeholder="CLIENT_KEY..."
          error={fieldErrors?.key?.message}
          {...register(`envVars.${index}.key`)}
        />
        <FormInput
          label="Value"
          className="flex-1 [&_input]:font-mono"
          placeholder="value"
          error={fieldErrors?.value?.message}
          {...register(`envVars.${index}.value`)}
        />
        {!isOnly && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="size-9 shrink-0 px-0 justify-center text-gray-11 hover:text-gray-12 hover:bg-grayA-3 rounded-lg mt-6.5"
            onClick={() => onRemove(index)}
          >
            <Trash iconSize="sm-regular" />
          </Button>
        )}
      </div>

      <details className="group">
        <summary className="w-fit text-[13px] text-gray-11 hover:text-gray-12 transition-colors cursor-pointer list-none [&::-webkit-details-marker]:hidden flex items-center gap-1.5 group">
          <span className="group-open:hidden flex items-center gap-2">
            <Plus
              iconSize="sm-medium"
              className="text-gray-9 group-hover:text-gray-12 transition-colors"
            />
            Add Note
          </span>
          <span className="hidden group-open:inline">Note</span>
        </summary>
        <div className="pt-1.5">
          <FormInput
            className="[&_input]:text-sm"
            placeholder="Optional description for this variable..."
            {...register(`envVars.${index}.description`)}
          />
        </div>
      </details>
    </div>
  );
};
