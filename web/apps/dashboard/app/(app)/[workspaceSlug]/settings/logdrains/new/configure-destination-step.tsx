"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { emptyHeader, toHeaderRecord } from "../header-fields";
import { DestinationFields } from "./destination-fields";
import { DestinationStepContainer } from "./destination-step-container";
import { type FormValues, formSchema, type Kind } from "./form-schema";

export function ConfigureDestinationStep({ kind }: { kind: Kind }) {
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const utils = trpc.useUtils();
  const create = trpc.logdrain.create.useMutation();
  const {
    register,
    control,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isValid },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      kind,
      name: "",
      url: "",
      format: "json",
      headers: [{ ...emptyHeader }],
      dataset: "",
      token: "",
      startFrom: "now",
    },
    mode: "onChange",
  });
  const format = watch("format");
  const startFrom = watch("startFrom");

  const submit = handleSubmit(async (values) => {
    try {
      switch (values.kind) {
        case "http":
          await create.mutateAsync({
            name: values.name,
            kind: values.kind,
            stream: "audit_logs",
            startFrom: values.startFrom,
            config: {
              url: values.url,
              format: values.format,
              headers: toHeaderRecord(values.headers),
            },
          });
          break;
        case "axiom":
          await create.mutateAsync({
            name: values.name,
            kind: values.kind,
            stream: "audit_logs",
            startFrom: values.startFrom,
            config: { dataset: values.dataset.trim(), token: values.token },
          });
          break;
        default:
          throw new Error(`Unsupported log drain sink: ${values.kind satisfies never}`);
      }
      await utils.logdrain.list.invalidate();
      toast.success("Log drain created");
      router.replace(routes.settings.logdrains.list({ workspaceSlug: workspace.slug }));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to create log drain");
    }
  });

  return (
    <DestinationStepContainer>
      <div className="flex w-full flex-col items-center justify-center gap-4 rounded-lg border border-grayA-5 px-4 py-[18px]">
        <form onSubmit={submit} className="flex w-full flex-col gap-4">
          <FormInput
            requirement="required"
            label="Name"
            description="Shown in the log drain list."
            className="[&_input:first-of-type]:h-[36px]"
            autoFocus
            error={errors.name?.message}
            placeholder="Production audit logs"
            {...register("name")}
          />

          <fieldset className="flex flex-col gap-1.5">
            <legend className="text-[13px] text-gray-11">Start delivery from</legend>
            <span className="text-xs text-gray-9">
              Choose how far back Unkey sends retained audit logs.
            </span>
            <div className="flex w-fit rounded-lg border border-grayA-4 p-1">
              {(
                [
                  { value: "now", label: "New audit logs" },
                  { value: "beginning", label: "All retained audit logs" },
                ] as const
              ).map((option) => (
                <Button
                  type="button"
                  key={option.value}
                  size="sm"
                  variant={startFrom === option.value ? "primary" : "ghost"}
                  aria-pressed={startFrom === option.value}
                  onClick={() => setValue("startFrom", option.value, { shouldValidate: true })}
                >
                  {option.label}
                </Button>
              ))}
            </div>
          </fieldset>

          <DestinationFields
            kind={kind}
            format={format}
            setFormat={(value) => setValue("format", value, { shouldValidate: true })}
            register={register}
            control={control}
            errors={errors}
          />

          <Button
            type="submit"
            variant="primary"
            size="xlg"
            disabled={create.isLoading || !isValid}
            loading={create.isLoading}
            className="mt-2 w-full rounded-lg"
          >
            Create log drain
          </Button>
        </form>
      </div>
    </DestinationStepContainer>
  );
}
