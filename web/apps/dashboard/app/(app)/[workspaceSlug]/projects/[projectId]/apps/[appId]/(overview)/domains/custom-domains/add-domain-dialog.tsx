"use client";

import type { Environment } from "@/lib/collections/deploy/environments";
import { Button, DialogContainer, FormInput, FormSelect } from "@unkey/ui";
import { Controller } from "react-hook-form";
import { EnvironmentLabel } from "../../../../_components/environment-label";
import { type AddDomainOutcome, useAddDomain } from "./use-add-domain";

type AddDomainDialogProps = {
  isOpen: boolean;
  onSettled: (outcome: AddDomainOutcome) => void;
  environments: Environment[];
  defaultEnvironmentId: string;
};

export function AddDomainDialog({
  isOpen,
  onSettled,
  environments,
  defaultEnvironmentId,
}: AddDomainDialogProps) {
  const { form, submit, cancel } = useAddDomain({ defaultEnvironmentId, onSettled });
  const {
    control,
    register,
    clearErrors,
    watch,
    formState: { isSubmitting, errors },
  } = form;

  const close = () => {
    if (isSubmitting) {
      return;
    }
    cancel();
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={(open) => {
        if (!open) {
          close();
        }
      }}
      preventOutsideClose={isSubmitting}
      title="Add custom domain"
      subTitle="The domain serves the environment you choose."
      footer={
        <div className="flex w-full items-center justify-between gap-3">
          <Button variant="outline" size="md" onClick={close} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button
            type="submit"
            form="add-custom-domain"
            variant="primary"
            size="md"
            className="px-3"
            disabled={watch("domain").trim() === ""}
            loading={isSubmitting}
          >
            Add domain
          </Button>
        </div>
      }
    >
      <form id="add-custom-domain" onSubmit={submit} className="flex flex-col gap-5">
        <FormInput
          data-1p-ignore
          label="Domain"
          placeholder="api.example.com"
          className="[&_input]:font-mono"
          error={errors.domain?.message}
          {...register("domain", { onChange: () => clearErrors("domain") })}
        />
        <Controller
          control={control}
          name="environmentId"
          render={({ field }) => (
            <FormSelect
              label="Environment"
              className="w-60"
              placeholder="Environment"
              error={errors.environmentId?.message}
              value={field.value}
              onValueChange={field.onChange}
              options={environments.map((env) => ({
                value: env.id,
                label: <EnvironmentLabel environment={env} className="text-sm text-gray-12" />,
              }))}
            />
          )}
        />
      </form>
    </DialogContainer>
  );
}
