import { UsageSetup } from "@/app/(app)/apis/[apiId]/_components/create-key/components/credits-setup";
import { ExpirationSetup } from "@/app/(app)/apis/[apiId]/_components/create-key/components/expiration-setup";
import { GeneralSetup } from "@/app/(app)/apis/[apiId]/_components/create-key/components/general-setup";
import {
  EXAMPLE_JSON,
  MetadataSetup,
} from "@/app/(app)/apis/[apiId]/_components/create-key/components/metadata-setup";
import { RatelimitSetup } from "@/app/(app)/apis/[apiId]/_components/create-key/components/ratelimit-setup";
import {
  type FormValues,
  formSchema,
} from "@/app/(app)/apis/[apiId]/_components/create-key/create-key.schema";
import { getDefaultValues } from "@/app/(app)/apis/[apiId]/_components/create-key/create-key.utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { CalendarClock, ChartPie, Code, Gauge, Key2, StackPerspective2 } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { addDays } from "date-fns";
import { useRef } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { ExpandableSettings } from "../components/expandable-settings";
import type { OnboardingStep } from "../components/onboarding-wizard";

const apiName = z.object({
  apiName: z.string().trim().min(3, "API name must be at least 3 characters long").max(50),
});

const extendedFormSchema = formSchema.and(apiName);

export const useKeyCreationStep = (): OnboardingStep => {
  const formRef = useRef<HTMLFormElement>(null);

  const methods = useForm<FormValues & { apiName: string }>({
    resolver: zodResolver(extendedFormSchema),
    mode: "onChange",
    shouldFocusError: true,
    shouldUnregister: true,
    defaultValues: getDefaultValues(),
  });

  const {
    handleSubmit,
    register,
    watch,
    formState: { errors },
  } = methods;
  const onSubmit = async (data: FormValues) => {
    console.info("DATA", data);
    try {
    } catch {
      // `useCreateKey` already shows a toast, but we still need to
      // prevent unhandled‐rejection noise in the console.
    }
  };

  const apiNameValue = watch("apiName");
  return {
    name: "API key",
    icon: <StackPerspective2 size="sm-regular" className="text-gray-11" />,
    body: (
      <div className="relative">
        <FormProvider {...methods}>
          <form id="new-key-form" onSubmit={handleSubmit(onSubmit)} ref={formRef}>
            <div className="flex flex-col px-1">
              <FormInput
                {...register("apiName")}
                error={errors.apiName?.message}
                placeholder="Enter API key name"
                description="This is just a human readable name for you and not visible to anyone else"
                label="API key name"
                optional
                className="w-full"
              />
              <div className="mt-8" />
              <div className="text-gray-11 text-[13px] leading-6">
                Fine-tune your API key by enabling additional options
              </div>
              <div className="mt-5" />
              <div className="flex flex-col gap-3 w-full">
                <ExpandableSettings
                  disabled={!apiNameValue}
                  icon={<Key2 className="text-gray-9 flex-shrink-0" size="sm-regular" />}
                  title="General Setup"
                  description="Configure basic API key settings like prefix, byte length, and External ID"
                >
                  <GeneralSetup />
                </ExpandableSettings>

                <ExpandableSettings
                  disabled={!apiNameValue}
                  icon={<Gauge className="text-gray-9 flex-shrink-0" size="sm-regular" />}
                  title="Ratelimit"
                  description="Set request limits per time window to control API usage frequency"
                  defaultChecked={methods.watch("ratelimit.enabled")}
                  onCheckedChange={(checked) => {
                    methods.setValue("ratelimit.enabled", checked);
                    methods.trigger("ratelimit");
                  }}
                >
                  {(enabled) => <RatelimitSetup overrideEnabled={enabled} />}
                </ExpandableSettings>

                <ExpandableSettings
                  disabled={!apiNameValue}
                  icon={<ChartPie className="text-gray-9 flex-shrink-0" size="sm-regular" />}
                  title="Credits"
                  description="Set usage limits based on credits or quota to control consumption"
                  defaultChecked={methods.watch("limit.enabled")}
                  onCheckedChange={(checked) => {
                    methods.setValue("limit.enabled", checked);
                    methods.trigger("limit");
                  }}
                >
                  {(enabled) => <UsageSetup overrideEnabled={enabled} />}
                </ExpandableSettings>

                <ExpandableSettings
                  disabled={!apiNameValue}
                  icon={<CalendarClock className="text-gray-9 flex-shrink-0" size="sm-regular" />}
                  title="Expiration"
                  description="Set when this API key should automatically expire and become invalid"
                  defaultChecked={methods.watch("expiration.enabled")}
                  onCheckedChange={(checked) => {
                    methods.setValue("expiration.enabled", checked);
                    const currentExpiryDate = methods.getValues("expiration.data");
                    // Set default expiry date (1 day) when enabling if not already set
                    if (checked && !currentExpiryDate) {
                      methods.setValue("expiration.data", addDays(new Date(), 1));
                    }
                    methods.trigger("expiration");
                  }}
                >
                  {(enabled) => <ExpirationSetup overrideEnabled={enabled} />}
                </ExpandableSettings>

                <ExpandableSettings
                  disabled={!apiNameValue}
                  icon={<Code className="text-gray-9 flex-shrink-0" size="sm-regular" />}
                  title="Metadata"
                  description="Add custom key-value pairs to store additional information with your API key"
                  defaultChecked={methods.watch("metadata.enabled")}
                  onCheckedChange={(checked) => {
                    methods.setValue("metadata.enabled", checked);
                    const currentMetadata = methods.getValues("metadata.data");
                    if (checked && !currentMetadata) {
                      methods.setValue("metadata.data", JSON.stringify(EXAMPLE_JSON, null, 2));
                    }

                    methods.trigger("metadata");
                  }}
                >
                  {(enabled) => <MetadataSetup overrideEnabled={enabled} />}
                </ExpandableSettings>
              </div>
            </div>
          </form>
        </FormProvider>
      </div>
    ),
    kind: "non-required" as const,
    buttonText: "Continue",
    description: "Setup your API key with extended configurations",
    onStepNext: () => {
      formRef.current?.requestSubmit();
    },
    onStepBack: () => {
      console.info("Going back from workspace step");
    },
  };
};
