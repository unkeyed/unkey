"use client";

import { OnboardingStepContainer } from "@/components/onboarding/step-container";
import { OnboardingStepHeader } from "@/components/onboarding/step-header";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Earth, Plus, Trash } from "@unkey/icons";
import {
  Button,
  FormInput,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
  StepWizard,
  toast,
  useStepWizard,
} from "@unkey/ui";
import { useRouter } from "next/navigation";
import { type ReactNode, useState } from "react";
import {
  type Control,
  type FieldErrors,
  type UseFormRegister,
  useFieldArray,
  useForm,
} from "react-hook-form";
import { z } from "zod";
import { AxiomLogo } from "../axiom-logo";
import { emptyHeader, headerFieldsSchema, toHeaderRecord } from "../header-fields";

type Kind = "http" | "axiom";
type HttpFormat = "json" | "ndjson";

const httpsUrlSchema = z
  .string()
  .url("Enter a valid URL")
  .superRefine((value, context) => {
    if (!URL.canParse(value)) {
      return;
    }
    const url = new URL(value);
    if (url.protocol !== "https:") {
      context.addIssue({ code: "custom", message: "URL must use HTTPS" });
    }
    if (url.username !== "" || url.password !== "") {
      context.addIssue({ code: "custom", message: "URL must not contain credentials" });
    }
  });

const formSchema = z
  .object({
    kind: z.enum(["http", "axiom"]),
    name: z.string().trim().min(1, "Enter a name").max(128, "Name must be 128 characters or less"),
    url: z.string(),
    format: z.enum(["json", "ndjson"]),
    headers: headerFieldsSchema,
    dataset: z.string(),
    token: z.string(),
    startFrom: z.enum(["now", "beginning"]),
  })
  .superRefine((values, context) => {
    switch (values.kind) {
      case "http": {
        const url = httpsUrlSchema.safeParse(values.url);
        if (!url.success) {
          context.addIssue({
            code: "custom",
            path: ["url"],
            message: url.error.issues[0]?.message ?? "Enter a valid HTTPS URL",
          });
        }
        break;
      }
      case "axiom":
        if (values.dataset.trim() === "") {
          context.addIssue({ code: "custom", path: ["dataset"], message: "Enter a dataset" });
        }
        if (values.token.trim() === "") {
          context.addIssue({ code: "custom", path: ["token"], message: "Enter a token" });
        }
        break;
      default:
        throw new Error(`Unsupported log drain sink: ${values.kind satisfies never}`);
    }
  });

type FormValues = z.infer<typeof formSchema>;

const OPTIONS: Array<{
  kind: Kind;
  title: string;
  description: string;
  icon: ReactNode;
}> = [
  {
    kind: "http",
    title: "HTTP",
    description: "Send audit logs to an HTTPS endpoint.",
    icon: <Earth className="size-[18px] text-gray-12" iconSize="md-medium" />,
  },
  {
    kind: "axiom",
    title: "Axiom",
    description: "Send audit logs to an Axiom dataset.",
    icon: <AxiomLogo className="size-[18px] text-gray-12" />,
  },
];

function sinkName(kind: Kind): string {
  switch (kind) {
    case "http":
      return "HTTP";
    case "axiom":
      return "Axiom";
    default:
      throw new Error(`Unsupported log drain sink: ${kind satisfies never}`);
  }
}

export default function NewLogdrainPage() {
  const [kind, setKind] = useState<Kind | null>(null);

  return (
    <StepWizard.Root>
      <StepWizard.Step id="choose-destination" label="Choose destination">
        <OnboardingStepContainer>
          <OnboardingStepHeader
            title="Create log drain"
            showIconRow
            subtitle="Choose where to send audit logs."
          />
          <ChooseDestinationStep onSelect={setKind} />
        </OnboardingStepContainer>
      </StepWizard.Step>
      <StepWizard.Step id="configure-destination" label="Configure destination">
        {kind ? (
          <OnboardingStepContainer>
            <OnboardingStepHeader
              title={`Configure ${sinkName(kind)} destination`}
              subtitle="Enter the destination details."
              allowBack
            />
            <ConfigureDestinationStep key={kind} kind={kind} />
          </OnboardingStepContainer>
        ) : null}
      </StepWizard.Step>
    </StepWizard.Root>
  );
}

function ChooseDestinationStep({ onSelect }: { onSelect: (kind: Kind) => void }) {
  const { next } = useStepWizard();

  const select = (kind: Kind) => {
    onSelect(kind);
    next();
  };

  return (
    <div className="flex w-[600px] max-w-[calc(100vw-2rem)] flex-col gap-3">
      {OPTIONS.map((option) => {
        return (
          <Item key={option.kind} variant="outline" className="px-4 py-[18px]">
            <ItemMedia className="size-8 rounded-[10px] ring-1 ring-grayA-4 shadow-sm shadow-grayA-8/20 dark:shadow-none">
              {option.icon}
            </ItemMedia>
            <ItemContent>
              <ItemTitle>{option.title}</ItemTitle>
              <ItemDescription>{option.description}</ItemDescription>
            </ItemContent>
            <ItemActions>
              <Button
                variant="outline"
                className="rounded-lg border-grayA-4 shadow-sm transition-[background-color,box-shadow] hover:bg-grayA-2 hover:shadow-md"
                onClick={() => select(option.kind)}
              >
                Use {option.title}
              </Button>
            </ItemActions>
          </Item>
        );
      })}
    </div>
  );
}

function ConfigureDestinationStep({ kind }: { kind: Kind }) {
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
    <div className="flex w-[600px] max-w-[calc(100vw-2rem)] flex-col items-center">
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
    </div>
  );
}

type DestinationFieldsProps = {
  register: UseFormRegister<FormValues>;
  errors: FieldErrors<FormValues>;
};

function DestinationFields({
  kind,
  format,
  setFormat,
  register,
  control,
  errors,
}: DestinationFieldsProps & {
  kind: Kind;
  control: Control<FormValues>;
  format: HttpFormat;
  setFormat: (format: HttpFormat) => void;
}) {
  switch (kind) {
    case "http":
      return (
        <HttpFields
          format={format}
          setFormat={setFormat}
          register={register}
          control={control}
          errors={errors}
        />
      );
    case "axiom":
      return <AxiomFields register={register} errors={errors} />;
    default:
      throw new Error(`Unsupported log drain sink: ${kind satisfies never}`);
  }
}

function HttpFields({
  format,
  setFormat,
  register,
  control,
  errors,
}: DestinationFieldsProps & {
  control: Control<FormValues>;
  format: HttpFormat;
  setFormat: (format: HttpFormat) => void;
}) {
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

function AxiomFields({ register, errors }: DestinationFieldsProps) {
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
