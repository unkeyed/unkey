import { UsageSetup } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/credits-setup";
import { ExpirationSetup } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/expiration-setup";
import { GeneralSetup } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/general-setup";
import {
  EXAMPLE_JSON,
  MetadataSetup,
} from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/metadata-setup";
import { RatelimitSetup } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/ratelimit-setup";
import {
  type FormValues,
  formSchema,
} from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/create-key.schema";
import {
  formValuesToApiInput,
  getDefaultValues,
} from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/create-key.utils";
import { useTRPC } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { CalendarClock, ChartPie, Code, Gauge, Key2, StackPerspective2 } from "@unkey/icons";
import { FormInput, toast } from "@unkey/ui";
import { addDays } from "date-fns";
import { useRouter, useSearchParams } from "next/navigation";
import { useRef, useState, useTransition } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { ExpandableSettings } from "../components/expandable-settings";
import type { OnboardingStep } from "../components/onboarding-wizard";
import { API_ID_PARAM, KEY_PARAM } from "../constants";

import { useMutation } from "@tanstack/react-query";

const extendedFormSchema = formSchema.and(
  z.object({
    apiName: z
      .string()
      .trim()
      .min(3, "API name must be at least 3 characters long")
      .max(50, "API name must not exceed 50 characters"),
  }),
);
type Props = {
  // Move to the next step
  advance: () => void;
};
export const useKeyCreationStep = (props: Props): OnboardingStep => {
  const trpc = useTRPC();
  const [apiCreated, setApiCreated] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);
  const router = useRouter();
  const searchParams = useSearchParams();
  const [isPending, startTransition] = useTransition();

  const createApiAndKey = useMutation(trpc.workspace.onboarding.mutationOptions({
    onSuccess: (data) => {
      setApiCreated(true);
      startTransition(() => {
        const params = new URLSearchParams(searchParams?.toString());
        params.set(API_ID_PARAM, data.apiId);
        params.set(KEY_PARAM, data.key);
        router.push(`?${params.toString()}`);
      });
    },
    onError: (error) => {
      console.error("Failed to create API and key:", error);

      if (error.data?.code === "NOT_FOUND") {
        // In case users try to feed tRPC with weird workspaceId or non existing one
        toast.error("Invalid workspace. Please go back and create a new workspace.");
      } else {
        toast.error(`Failed to create API and key: ${error.message}`);
      }
    },
  }));

  const methods = useForm<FormValues & { apiName: string }>({
    resolver: zodResolver(extendedFormSchema),
    mode: "onChange",
    shouldFocusError: true,
    shouldUnregister: true,
    defaultValues: {
      ...getDefaultValues(),
    },
  });

  const {
    handleSubmit,
    register,
    watch,
    formState: { errors },
  } = methods;

  const onSubmit = async (data: FormValues & { apiName: string }) => {
    try {
      const keyInput = formValuesToApiInput(data, ""); // Empty keyAuthId since we'll create it
      const { keyAuthId, ...keyInputWithoutAuthId } = keyInput; // Remove keyAuthId

      const submitData = {
        apiName: data.apiName,
        ...keyInputWithoutAuthId,
      };

      await createApiAndKey.mutateAsync(submitData);
      props.advance();
    } catch (error) {
      console.error("Submit error:", error);
    }
  };

  const apiNameValue = watch("apiName");
  const isFormReady = Boolean(apiNameValue);
  const isLoading = createApiAndKey.isPending || isPending;

  const tooltipContent = apiCreated
    ? "API already created - settings cannot be modified"
    : isFormReady
      ? "Settings are currently disabled"
      : "You need to have a valid API name";

  return {
    name: "API key",
    icon: <StackPerspective2 iconSize="sm-regular" className="text-gray-11" />,
    body: (
      <div className="relative">
        <FormProvider {...methods}>
          <form id="new-key-form" onSubmit={handleSubmit(onSubmit)} ref={formRef}>
            <div className="flex flex-col px-1">
              <FormInput
                {...register("apiName")}
                error={errors.apiName?.message}
                placeholder="Enter API name"
                description="Choose a name for your API that helps you identify it"
                label="API name"
                className="w-full"
                disabled={isLoading || apiCreated}
              />

              <div className="mt-8" />

              <div className="text-gray-11 text-[13px] leading-6">
                Fine-tune your API key by enabling additional options
              </div>

              <div className="mt-5" />

              <div className="flex flex-col gap-3 w-full">
                <ExpandableSettings
                  disabled={!isFormReady || isLoading || apiCreated}
                  disabledTooltip={tooltipContent}
                  icon={<Key2 className="text-gray-9 flex-shrink-0" iconSize="sm-regular" />}
                  title="General Setup"
                  description="Configure basic API key settings like prefix, byte length, and External ID"
                >
                  <GeneralSetup />
                </ExpandableSettings>

                <ExpandableSettings
                  disabled={!isFormReady || isLoading || apiCreated}
                  disabledTooltip={tooltipContent}
                  icon={<Gauge className="text-gray-9 flex-shrink-0" iconSize="sm-regular" />}
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
                  disabled={!isFormReady || isLoading || apiCreated}
                  disabledTooltip={tooltipContent}
                  icon={<ChartPie className="text-gray-9 flex-shrink-0" iconSize="sm-regular" />}
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
                  disabled={!isFormReady || isLoading || apiCreated}
                  disabledTooltip={tooltipContent}
                  icon={
                    <CalendarClock className="text-gray-9 flex-shrink-0" iconSize="sm-regular" />
                  }
                  title="Expiration"
                  description="Set when this API key should automatically expire and become invalid"
                  defaultChecked={methods.watch("expiration.enabled")}
                  onCheckedChange={(checked) => {
                    methods.setValue("expiration.enabled", checked);
                    const currentExpiryDate = methods.getValues("expiration.data");
                    if (checked && !currentExpiryDate) {
                      methods.setValue("expiration.data", addDays(new Date(), 1));
                    }
                    methods.trigger("expiration");
                  }}
                >
                  {(enabled) => <ExpirationSetup overrideEnabled={enabled} />}
                </ExpandableSettings>

                <ExpandableSettings
                  disabled={!isFormReady || isLoading || apiCreated}
                  disabledTooltip={tooltipContent}
                  icon={<Code className="text-gray-9 flex-shrink-0" iconSize="sm-regular" />}
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
    buttonText: apiCreated ? "Continue" : isLoading ? "Creating API & Key..." : "Create API & Key",
    description: apiCreated
      ? "API and key created successfully, continue to next step"
      : "Setup your API with an initial key and advanced configurations",
    onStepSkip: () => {
      router.push("/apis");
    },
    onStepNext: apiCreated
      ? () => true
      : () => {
        if (!isLoading) {
          formRef.current?.requestSubmit();
        }
        return false;
      },
    isLoading,
  };
};
