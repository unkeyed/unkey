import { cn } from "@/lib/utils";
import { ChevronDown, Eye, EyeSlash, Plus } from "@unkey/icons";
import {
  Button,
  FormCheckbox,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { memo, useState } from "react";
import {
  type Control,
  Controller,
  type UseFormRegister,
  type UseFormTrigger,
  useWatch,
} from "react-hook-form";
import { RemoveButton } from "../../shared/remove-button";
import type { EnvVarsFormValues } from "./schema";
import type { EnvVarItem } from "./utils";

type EnvVarRowProps = {
  index: number;
  isFirst: boolean;
  isOnly: boolean;
  keyError: string | undefined;
  environmentError: string | undefined;
  defaultEnvVars: EnvVarItem[];
  environments: { id: string; slug: string }[];
  control: Control<EnvVarsFormValues>;
  register: UseFormRegister<EnvVarsFormValues>;
  trigger: UseFormTrigger<EnvVarsFormValues>;
  onAdd: () => void;
  onRemove: (index: number) => void;
};

export const EnvVarRow = memo(function EnvVarRow({
  index,
  isFirst,
  isOnly,
  keyError,
  environmentError,
  defaultEnvVars,
  environments,
  control,
  register,
  trigger,
  onAdd,
  onRemove,
}: EnvVarRowProps) {
  const [isVisible, setIsVisible] = useState(false);

  // Watch this specific row's data - fixes index shift bug on delete
  const currentVar = useWatch({ control, name: `envVars.${index}` });
  const isSecret = currentVar?.secret ?? false;
  const isPreviouslyAdded = Boolean(
    currentVar?.id && defaultEnvVars.some((v) => v.id === currentVar.id && v.key !== ""),
  );

  const inputType = isPreviouslyAdded ? (isVisible ? "text" : "password") : "text";

  const eyeButton =
    isPreviouslyAdded && !isSecret ? (
      <button
        type="button"
        className="text-gray-9 hover:text-gray-11 transition-colors"
        onClick={() => setIsVisible((v) => !v)}
        tabIndex={-1}
      >
        {isVisible ? <EyeSlash iconSize="sm-regular" /> : <Eye iconSize="sm-regular" />}
      </button>
    ) : undefined;

  return (
    <div className="flex items-start gap-2">
      <div className="w-[120px] shrink-0">
        <Controller
          control={control}
          name={`envVars.${index}.environmentId`}
          render={({ field }) => (
            <Select
              value={field.value}
              onValueChange={(value) => {
                field.onChange(value);
                trigger("envVars");
              }}
              disabled={isPreviouslyAdded}
            >
              <SelectTrigger
                className={cn("h-9", environmentError && !isPreviouslyAdded && "border-error-9")}
                wrapperClassName="w-[120px]"
                rightIcon={<ChevronDown className="absolute right-3 size-3 opacity-70" />}
              >
                <SelectValue placeholder="Env" />
              </SelectTrigger>
              <SelectContent>
                {environments.map((env) => (
                  <SelectItem key={env.id} value={env.id}>
                    {env.slug}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        />
      </div>
      <FormInput
        className="flex-1 [&_input]:font-mono"
        placeholder="MY_VAR"
        disabled={isPreviouslyAdded && isSecret}
        error={keyError}
        {...register(`envVars.${index}.key`)}
      />
      <FormInput
        className="flex-1 [&_input]:font-mono"
        disabled={isPreviouslyAdded && isSecret}
        placeholder={isPreviouslyAdded && isSecret ? "sensitive" : "value"}
        type={inputType}
        rightIcon={eyeButton}
        {...register(`envVars.${index}.value`)}
      />
      <div className="bg-grayA-3 w-16 h-9 flex items-center justify-center rounded-lg">
        <Controller
          control={control}
          name={`envVars.${index}.secret`}
          render={({ field }) => (
            <FormCheckbox
              disabled={isPreviouslyAdded}
              className="bg-white data-[state=checked]:bg-white data-[state=unchecked]:bg-white rounded"
              size="lg"
              checked={field.value}
              onCheckedChange={field.onChange}
              name={field.name}
              ref={field.ref}
            />
          )}
        />
      </div>
      <div className="relative w-16 h-9 shrink-0">
        <RemoveButton
          onClick={() => onRemove(index)}
          className={cn(
            "absolute left-0 transition-opacity duration-150",
            isOnly ? "opacity-0 pointer-events-none" : "opacity-100",
          )}
        />
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className={cn(
            "absolute left-0 size-9 hover:bg-grayA-3 px-0 justify-center transition-all duration-150 rounded-lg",
            isOnly ? "translate-x-0" : "translate-x-9",
            isFirst ? "opacity-100" : "opacity-0 pointer-events-none",
          )}
          onClick={onAdd}
        >
          <Plus iconSize="sm-regular" />
        </Button>
      </div>
    </div>
  );
});
