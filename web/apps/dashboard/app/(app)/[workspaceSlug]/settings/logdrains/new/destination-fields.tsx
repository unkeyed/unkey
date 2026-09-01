"use client";

import { Plus, Trash } from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, FormInput } from "@unkey/ui";
import type { Control, FieldErrors, UseFormRegister } from "react-hook-form";
import { useFieldArray } from "react-hook-form";
import { emptyHeader } from "../header-fields";
import type { FormValues, HttpFormat, Kind } from "./form-schema";

type DestinationFieldsProps = {
  kind: Kind;
  register: UseFormRegister<FormValues>;
  control: Control<FormValues>;
  errors: FieldErrors<FormValues>;
  format: HttpFormat;
  setFormat: (format: HttpFormat) => void;
};

export function DestinationFields({
  kind,
  format,
  setFormat,
  register,
  control,
  errors,
}: DestinationFieldsProps) {
  return match(kind)
    .with("http", () => (
      <HttpFields
        format={format}
        setFormat={setFormat}
        register={register}
        control={control}
        errors={errors}
      />
    ))
    .with("axiom", () => <AxiomFields register={register} errors={errors} />)
    .exhaustive();
}

function HttpFields({
  format,
  setFormat,
  register,
  control,
  errors,
}: Omit<DestinationFieldsProps, "kind">) {
  const { fields, append, remove } = useFieldArray({ control, name: "headers" });

  return (
    <>
      <FormInput
        requirement="required"
        label="HTTPS endpoint"
        description="Unkey sends each audit log batch to this URL."
        className="[&_input:first-of-type]:h-[36px]"
        error={errors.url?.message}
        placeholder="https://example.com/audit"
        {...register("url")}
      />
      <fieldset className="flex flex-col gap-3">
        <legend className="text-[13px] text-gray-11">Headers</legend>
        <span className="-mt-2 text-xs text-gray-9">
          Optional. Unkey encrypts header values before storing them.
        </span>
        {fields.map((field, index) => (
          <div key={field.id} className="flex items-start gap-3">
            <FormInput
              label="Name"
              placeholder="Authorization"
              className="flex-1 [&_input:first-of-type]:h-[36px]"
              error={errors.headers?.[index]?.name?.message}
              {...register(`headers.${index}.name`)}
            />
            <FormInput
              label="Value"
              placeholder="Bearer …"
              className="flex-1 [&_input:first-of-type]:h-[36px]"
              error={errors.headers?.[index]?.value?.message}
              {...register(`headers.${index}.value`)}
            />
            {fields.length > 1 ? (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="mt-[26px] size-9 shrink-0 justify-center px-0 text-gray-11"
                aria-label={`Remove header ${index + 1}`}
                onClick={() => remove(index)}
              >
                <Trash iconSize="sm-regular" />
              </Button>
            ) : null}
          </div>
        ))}
        <Button
          type="button"
          variant="outline"
          className="w-fit"
          disabled={fields.length >= 32}
          onClick={() => append({ ...emptyHeader })}
        >
          <Plus iconSize="sm-regular" />
          Add header
        </Button>
      </fieldset>
      <fieldset className="flex flex-col gap-1.5">
        <legend className="text-[13px] text-gray-11">Body format</legend>
        <div className="flex w-fit rounded-lg border border-grayA-4 p-1">
          {(["json", "ndjson"] as const).map((value) => (
            <Button
              type="button"
              key={value}
              size="sm"
              variant={format === value ? "primary" : "ghost"}
              aria-pressed={format === value}
              onClick={() => setFormat(value)}
            >
              {value === "json" ? "JSON array" : "NDJSON"}
            </Button>
          ))}
        </div>
        <span className="text-xs text-gray-9">
          JSON sends an array of events. NDJSON sends one event per line.
        </span>
      </fieldset>
    </>
  );
}

function AxiomFields({ register, errors }: Pick<DestinationFieldsProps, "register" | "errors">) {
  return (
    <>
      <FormInput
        requirement="required"
        label="Dataset"
        description="The Axiom dataset that receives audit logs."
        className="[&_input:first-of-type]:h-[36px]"
        error={errors.dataset?.message}
        placeholder="audit-logs"
        {...register("dataset")}
      />
      <FormInput
        requirement="required"
        label="Token"
        description="Use an Axiom API token that can ingest data into this dataset."
        className="[&_input:first-of-type]:h-[36px]"
        type="password"
        error={errors.token?.message}
        autoComplete="off"
        {...register("token")}
      />
    </>
  );
}
