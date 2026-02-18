"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { Eye, EyeSlash, Nodes2, Plus, Trash } from "@unkey/icons";
import { Button, FormCheckbox, FormInput, toast } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useEffect, useState } from "react";
import {
  type Control,
  Controller,
  type UseFormRegister,
  useFieldArray,
  useForm,
  useWatch,
} from "react-hook-form";
import { FormSettingCard } from "../../shared/form-setting-card";
import { EMPTY_ROW, type EnvVarsFormValues, envVarsSchema } from "./schema";
import { useDropZone } from "./use-drop-zone";

export const EnvVars = () => {
  const defaultValues: EnvVarsFormValues = {
    envVars: [{ ...EMPTY_ROW }],
  };

  return <EnvVarsForm defaultValues={defaultValues} />;
};

type EnvVarsFormProps = {
  defaultValues: EnvVarsFormValues;
};

const EnvVarsForm: React.FC<EnvVarsFormProps> = ({ defaultValues }) => {
  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
    reset,
  } = useForm<EnvVarsFormValues>({
    resolver: zodResolver(envVarsSchema),
    mode: "onChange",
    defaultValues,
  });

  const { ref, isDragging } = useDropZone(reset);

  useEffect(() => {
    reset(defaultValues);
  }, [reset]);

  const { fields, append, remove } = useFieldArray({ control, name: "envVars" });

  const currentEnvVars = useWatch({ control, name: "envVars" });

  const hasChanges = JSON.stringify(currentEnvVars) !== JSON.stringify(defaultValues.envVars);

  const onSubmit = async (values: EnvVarsFormValues) => {
    // TODO: wire to trpc.deploy.environmentSettings.updateRuntime
    console.log("env vars:", values.envVars);
    toast.success("Environment variables saved");
  };

  const displayValue = (() => {
    const vars = defaultValues.envVars.filter((v) => v.key !== "");
    if (vars.length === 0) {
      return <span className="text-gray-11 font-normal">None</span>;
    }
    if (vars.length === 1) {
      return <span className="font-medium text-gray-12 font-mono text-xs">{vars[0].key}</span>;
    }
    return <span className="text-gray-11 font-normal">{vars.length} variables</span>;
  })();

  return (
    <FormSettingCard
      icon={<Nodes2 className="text-gray-12" iconSize="xl-medium" />}
      title="Environment Variables"
      description="Set environment variables available at runtime. Changes apply on next deploy."
      displayValue={displayValue}
      onSubmit={handleSubmit(onSubmit)}
      canSave={isValid && !isSubmitting && hasChanges}
      isSaving={isSubmitting}
      ref={ref}
      className={cn("relative", isDragging && "bg-primary/5")}
    >
      <div
        className={cn(
          "absolute inset-2 rounded-lg border-2 border-dotted pointer-events-none transition-colors",
          isDragging ? "border-successA-9" : "border-transparent",
        )}
      />
      <div className="flex flex-col gap-2 w-[520px]">
        <p className="text-xs text-gray-11 mb-2">
          Drag & drop your <span className="font-mono font-medium text-gray-12">.env</span> file or
          paste <span className="font-mono font-medium text-gray-12">env</span> vars (⌘V / Ctrl+V)
        </p>

        <div className="flex items-center gap-2">
          <span className="flex-1 text-[13px] text-gray-11">Key</span>
          <span className="flex-1 text-[13px] text-gray-11">Value</span>
          <span className="w-9 text-center text-[13px] text-gray-11">Secret</span>
          <div className="w-16 shrink-0" />
        </div>

        {fields.map((field, index) => {
          const isLast = index === fields.length - 1;
          const isOnly = fields.length === 1;
          const keyError = errors.envVars?.[index]?.key?.message;
          const isPreviouslyAdded =
            index < defaultValues.envVars.length && defaultValues.envVars[index].key !== "";

          return (
            <EnvVarRow
              key={field.id}
              index={index}
              isLast={isLast}
              isOnly={isOnly}
              keyError={keyError}
              isPreviouslyAdded={isPreviouslyAdded}
              control={control}
              register={register}
              onAdd={() => append({ ...EMPTY_ROW })}
              onRemove={() => remove(index)}
            />
          );
        })}
      </div>
    </FormSettingCard>
  );
};

type EnvVarRowProps = {
  index: number;
  isLast: boolean;
  isOnly: boolean;
  keyError: string | undefined;
  isPreviouslyAdded: boolean;
  control: Control<EnvVarsFormValues>;
  register: UseFormRegister<EnvVarsFormValues>;
  onAdd: () => void;
  onRemove: () => void;
};

const EnvVarRow: React.FC<EnvVarRowProps> = ({
  index,
  isLast,
  isOnly,
  keyError,
  isPreviouslyAdded,
  control,
  register,
  onAdd,
  onRemove,
}) => {
  const [isVisible, setIsVisible] = useState(false);

  const inputType = isPreviouslyAdded ? (isVisible ? "text" : "password") : "text";

  const eyeButton = isPreviouslyAdded ? (
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
      <FormInput
        className="flex-1 [&_input]:h-9 font-mono"
        placeholder="MY_VAR"
        error={keyError}
        {...register(`envVars.${index}.key`)}
      />
      <FormInput
        className="flex-1 [&_input]:h-9 font-mono"
        placeholder="value"
        type={inputType}
        rightIcon={eyeButton}
        {...register(`envVars.${index}.value`)}
      />
      <div className="bg-grayA-3 size-9 flex items-center justify-center rounded-lg">
        <Controller
          control={control}
          name={`envVars.${index}.secret`}
          render={({ field }) => (
            <FormCheckbox
              className="bg-white data-[state=checked]:bg-white data-[state=unchecked]:bg-white rounded"
              checked={field.value}
              onCheckedChange={field.onChange}
            />
          )}
        />
      </div>
      <div className="relative w-16 h-7 mt-1 shrink-0">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className={`absolute left-0 w-7 px-0 justify-center text-error-11 hover:text-error-11 transition-opacity duration-150 ${isOnly ? "opacity-0 pointer-events-none" : "opacity-100"}`}
          onClick={onRemove}
        >
          <Trash iconSize="sm-regular" />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className={`absolute left-0 w-7 px-0 justify-center transition-all duration-150 ${isOnly ? "translate-x-0" : "translate-x-9"} ${isLast ? "opacity-100" : "opacity-0 pointer-events-none"}`}
          onClick={onAdd}
        >
          <Plus iconSize="sm-regular" />
        </Button>
      </div>
    </div>
  );
};
