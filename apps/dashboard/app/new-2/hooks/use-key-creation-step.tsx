import { UsageSetup } from "@/app/(app)/apis/[apiId]/_components/create-key/components/credits-setup";
import { ExpirationSetup } from "@/app/(app)/apis/[apiId]/_components/create-key/components/expiration-setup";
import { GeneralSetup } from "@/app/(app)/apis/[apiId]/_components/create-key/components/general-setup";
import { MetadataSetup } from "@/app/(app)/apis/[apiId]/_components/create-key/components/metadata-setup";
import { RatelimitSetup } from "@/app/(app)/apis/[apiId]/_components/create-key/components/ratelimit-setup";
import {
  type FormValues,
  formSchema,
} from "@/app/(app)/apis/[apiId]/_components/create-key/create-key.schema";
import { getDefaultValues } from "@/app/(app)/apis/[apiId]/_components/create-key/create-key.utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { CalendarClock, ChartPie, Code, Gauge, Key2, StackPerspective2 } from "@unkey/icons";
import { FormInput } from "@unkey/ui";
import { useRef } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { ExpandableSettings } from "../components/expandable-settings";
import type { OnboardingStep } from "../components/onboarding-wizard";

export const useKeyCreationStep = (): OnboardingStep => {
  const formRef = useRef<HTMLFormElement>(null);
  const methods = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
    shouldFocusError: true,
    shouldUnregister: true,
    defaultValues: getDefaultValues(),
  });

  const { handleSubmit } = methods;

  const onSubmit = async (data: FormValues) => {
    console.info("DATA", data);
    try {
    } catch {
      // `useCreateKey` already shows a toast, but we still need to
      // prevent unhandled‐rejection noise in the console.
    }
  };

  return {
    name: "API key",
    icon: <StackPerspective2 size="sm-regular" className="text-gray-11" />,
    body: (
      <div className="relative">
        <FormProvider {...methods}>
          <form id="new-key-form" onSubmit={handleSubmit(onSubmit)} ref={formRef}>
            <div className="flex flex-col">
              <FormInput
                placeholder="Enter workspace name"
                label="Workspace name"
                optional
                className="w-full"
              />
              <div className="mt-8" />
              <div className="text-gray-11 text-[13px] leading-6">
                Fine-tune your API key by enabling additional options
              </div>
              <div className="mt-5" />

              <div className="flex flex-col gap-3 w-full h-">
                <ExpandableSettings
                  icon={<Key2 className="text-gray-9 flex-shrink-0" size="sm-regular" />}
                  title="General Setup"
                >
                  <GeneralSetup />
                </ExpandableSettings>
                <ExpandableSettings
                  icon={<Gauge className="text-gray-9 flex-shrink-0" size="sm-regular" />}
                  title="Ratelimit"
                  defaultChecked={methods.watch("ratelimit.enabled")}
                  onCheckedChange={(checked) => {
                    methods.setValue("ratelimit.enabled", checked);
                    methods.trigger("ratelimit");
                  }}
                >
                  {(enabled) => <RatelimitSetup overrideEnabled={enabled} />}
                </ExpandableSettings>{" "}
                <ExpandableSettings
                  icon={<ChartPie className="text-gray-9 flex-shrink-0" size="sm-regular" />}
                  title="Credits"
                  onCheckedChange={(checked) => {
                    methods.setValue("limit.enabled", checked);
                    methods.trigger("limit");
                  }}
                >
                  {(enabled) => <UsageSetup overrideEnabled={enabled} />}
                </ExpandableSettings>
                <ExpandableSettings
                  icon={<CalendarClock className="text-gray-9 flex-shrink-0" size="sm-regular" />}
                  title="Expiration"
                  onCheckedChange={(checked) => {
                    methods.setValue("expiration.enabled", checked);
                    methods.trigger("expiration");
                  }}
                >
                  {(enabled) => <ExpirationSetup overrideEnabled={enabled} />}
                </ExpandableSettings>
                <ExpandableSettings
                  icon={<Code className="text-gray-9 flex-shrink-0" size="sm-regular" />}
                  title="Metadata"
                  onCheckedChange={(checked) => {
                    methods.setValue("metadata.enabled", checked);
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
