"use client";

import type { Environment } from "@/lib/collections/deploy/environments";
import { IconChevronDownOutline12 } from "@unkey/icons";
import {
  Button,
  DialogContainer,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { Controller } from "react-hook-form";
import { EnvironmentLabel } from "../../../components/environment-label";
import { useAddDomain } from "./use-add-domain";

type AddDomainDialogProps = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onLimitReached: (message: string) => void;
  onAdded: (domain: string) => void;
  environments: Environment[];
  defaultEnvironmentId: string;
};

export function AddDomainDialog({
  isOpen,
  onOpenChange,
  onLimitReached,
  onAdded,
  environments,
  defaultEnvironmentId,
}: AddDomainDialogProps) {
  const { form, submit, close } = useAddDomain({
    defaultEnvironmentId,
    onOpenChange,
    onAdded,
    onLimitReached,
  });
  const {
    control,
    register,
    clearErrors,
    watch,
    formState: { isSubmitting, errors },
  } = form;

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={(open) => {
        if (!open) {
          close();
        }
      }}
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
        <div className="flex flex-col gap-1.5">
          <span className="text-sm font-medium text-gray-12">Environment</span>
          <Controller
            control={control}
            name="environmentId"
            render={({ field }) => {
              const selected = environments.find((e) => e.id === field.value);
              return (
                <Select value={field.value} onValueChange={field.onChange}>
                  <SelectTrigger
                    wrapperClassName="w-60"
                    aria-label="Environment"
                    variant={errors.environmentId ? "error" : "default"}
                    rightIcon={<IconChevronDownOutline12 className="absolute right-3 opacity-70" />}
                  >
                    <SelectValue placeholder="Environment">
                      {selected ? (
                        <EnvironmentLabel environment={selected} className="text-gray-12" />
                      ) : null}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {environments.map((env) => (
                      <SelectItem key={env.id} value={env.id}>
                        <EnvironmentLabel environment={env} className="text-gray-12" />
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              );
            }}
          />
        </div>
      </form>
    </DialogContainer>
  );
}
