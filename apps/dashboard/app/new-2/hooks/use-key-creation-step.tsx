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
import {
  formValuesToApiInput,
  getDefaultValues,
} from "@/app/(app)/apis/[apiId]/_components/create-key/create-key.utils";
import { setCookie } from "@/lib/auth/cookies";
import { UNKEY_SESSION_COOKIE } from "@/lib/auth/types";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { CalendarClock, ChartPie, Code, Gauge, Key2, StackPerspective2 } from "@unkey/icons";
import { FormInput, toast } from "@unkey/ui";
import { addDays } from "date-fns";
import { useSearchParams } from "next/navigation";
import { useRef } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { ExpandableSettings } from "../components/expandable-settings";
import type { OnboardingStep } from "../components/onboarding-wizard";

const extendedFormSchema = formSchema.and(
  z.object({
    apiName: z
      .string()
      .trim()
      .min(3, "API name must be at least 3 characters long")
      .max(50, "API name must not exceed 50 characters"),
  }),
);

export const useKeyCreationStep = (): OnboardingStep => {
  const formRef = useRef<HTMLFormElement>(null);
  const searchParams = useSearchParams();
  const workspaceName = searchParams?.get("workspaceName") || "";

  const createWorkspaceWithApiAndKey = trpc.workspace.onboarding.useMutation({
    onSuccess: (data) => {
      console.info("Successfully created workspace, API and key:", data);
      switchOrgMutation.mutate(data.organizationId);
    },
    onError: (error) => {
      console.error("Failed to create workspace, API and key:", error);
      // Handle error - show toast notification
    },
  });

  const methods = useForm<FormValues & { apiName: string }>({
    resolver: zodResolver(extendedFormSchema),
    mode: "onChange",
    shouldFocusError: true,
    shouldUnregister: true,
    defaultValues: {
      ...getDefaultValues(),
      apiName: "",
    },
  });

  const {
    handleSubmit,
    register,
    watch,
    formState: { errors },
  } = methods;

  const switchOrgMutation = trpc.user.switchOrg.useMutation({
    onSuccess: async (sessionData) => {
      if (!sessionData.expiresAt) {
        console.error("Missing session data: ", sessionData);
        toast.error(`Failed to switch organizations: ${sessionData.error}`);
        return;
      }

      await setCookie({
        name: UNKEY_SESSION_COOKIE,
        value: sessionData.token,
        options: {
          httpOnly: true,
          secure: true,
          sameSite: "strict",
          path: "/",
          maxAge: Math.floor((sessionData.expiresAt.getTime() - Date.now()) / 1000),
        },
      });
    },
    onError: (error) => {
      toast.error(`Failed to load new workspace: ${error.message}`);
    },
  });

  const onSubmit = async (data: FormValues & { apiName: string }) => {
    console.info("Submitting onboarding data:", data);

    if (!workspaceName) {
      console.error("Workspace name not found in URL parameters");
      return;
    }

    try {
      const keyInput = formValuesToApiInput(data, ""); // Empty keyAuthId since we'll create it
      const { keyAuthId, ...keyInputWithoutAuthId } = keyInput; // Remove keyAuthId

      const submitData = {
        workspaceName,
        apiName: data.apiName,
        ...keyInputWithoutAuthId,
      };

      await createWorkspaceWithApiAndKey.mutateAsync(submitData);
    } catch (error) {
      console.error("Submit error:", error);
    }
  };

  const apiNameValue = watch("apiName");
  const isFormReady = Boolean(workspaceName && apiNameValue);

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
                placeholder="Enter API name"
                description="Choose a name for your API that helps you identify it"
                label="API name"
                className="w-full"
                disabled={!workspaceName || createWorkspaceWithApiAndKey.isLoading}
              />

              <div className="mt-8" />

              <div className="text-gray-11 text-[13px] leading-6">
                Fine-tune your API key by enabling additional options
              </div>

              <div className="mt-5" />

              <div className="flex flex-col gap-3 w-full">
                <ExpandableSettings
                  disabled={!isFormReady || createWorkspaceWithApiAndKey.isLoading}
                  icon={<Key2 className="text-gray-9 flex-shrink-0" size="sm-regular" />}
                  title="General Setup"
                  description="Configure basic API key settings like prefix, byte length, and External ID"
                >
                  <GeneralSetup />
                </ExpandableSettings>

                <ExpandableSettings
                  disabled={!isFormReady || createWorkspaceWithApiAndKey.isLoading}
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
                  disabled={!isFormReady || createWorkspaceWithApiAndKey.isLoading}
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
                  disabled={!isFormReady || createWorkspaceWithApiAndKey.isLoading}
                  icon={<CalendarClock className="text-gray-9 flex-shrink-0" size="sm-regular" />}
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
                  disabled={!isFormReady || createWorkspaceWithApiAndKey.isLoading}
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
    buttonText: createWorkspaceWithApiAndKey.isLoading
      ? "Creating workspace..."
      : workspaceName
        ? "Create API & Key"
        : "Go Back",
    description: "Setup your API with an initial key and advanced configurations",
    onStepNext: () => {
      if (!workspaceName) {
        // Handle going back if workspace name is missing
        return;
      }
      formRef.current?.requestSubmit();
    },
    onStepBack: () => {
      console.info("Going back from API key creation step");
    },
  };
};
