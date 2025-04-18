"use client";
import { type NavItem, NavigableDialog } from "@/components/dialog-container/navigable-dialog";
import { zodResolver } from "@hookform/resolvers/zod";
import { CalendarClock, ChartPie, Code, Gauge, Key2 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { UsageSetup } from "./components/credits-setup";
import { ExpirationSetup } from "./components/expiration-setup";
import { GeneralSetup } from "./components/general-setup";
import { MetadataSetup } from "./components/metadata-setup";
import { RatelimitSetup } from "./components/ratelimit-setup";
import { SectionLabel } from "./components/section-label";
import {
  getDefaultValues,
  getFieldsFromSchema,
  isFeatureEnabled,
  processFormData,
  sectionSchemaMap,
} from "./form-utils";
import { type FormValues, formSchema } from "./schema";
import type { SectionName, SectionState } from "./types";

const DEFAULT_STEP_STATES: Record<SectionName, SectionState> = {
  general: "initial",
  metadata: "initial",
  expiration: "initial",
  ratelimit: "initial",
  credits: "initial",
};

export const CreateKeyDialog = () => {
  const [validSteps, setValidSteps] =
    useState<Record<SectionName, SectionState>>(DEFAULT_STEP_STATES);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);

  const methods = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
    shouldFocusError: true,
    shouldUnregister: true,
    defaultValues: getDefaultValues(),
  });

  const onSubmit = (data: FormValues) => {
    processFormData(data);
    methods.reset(getDefaultValues());
    setIsSettingsOpen(false);
    setValidSteps(DEFAULT_STEP_STATES);
  };

  const handleSectionNavigation = async (fromId: SectionName) => {
    // Skip validation for non-existent sections
    if (!sectionSchemaMap[fromId]) {
      return true;
    }

    // Skip validation if the feature is not enabled
    if (fromId !== "general" && !isFeatureEnabled(fromId, methods.getValues())) {
      setValidSteps((prevState) => ({
        ...prevState,
        [fromId]: "initial",
      }));
      return true;
    }

    // Get the schema for the section
    const schema = sectionSchemaMap[fromId];
    // Get fields from the schema
    const fieldsToValidate = getFieldsFromSchema(schema);
    // Skip validation if no fields to validate
    if (fieldsToValidate.length === 0) {
      return true;
    }

    // Trigger validation for the fields
    const result = await methods.trigger(fieldsToValidate as any);
    setValidSteps((prevState) => ({
      ...prevState,
      [fromId]: result ? "valid" : "invalid",
    }));
    // Always allow navigation
    return true;
  };

  const settingsNavItems: NavItem<SectionName>[] = [
    {
      id: "general",
      label: <SectionLabel label="General Setup" validState={validSteps.general} />,
      icon: Key2,
      content: <GeneralSetup />,
    },
    {
      id: "ratelimit",
      label: <SectionLabel label="Ratelimit" validState={validSteps.ratelimit} />,
      icon: Gauge,
      content: <RatelimitSetup />,
    },
    {
      id: "credits",
      label: <SectionLabel label="Credits" validState={validSteps.credits} />,
      icon: ChartPie,
      content: <UsageSetup />,
    },
    {
      id: "expiration",
      label: <SectionLabel label="Expiration" validState={validSteps.expiration} />,
      icon: CalendarClock,
      content: <ExpirationSetup />,
    },
    {
      id: "metadata",
      label: <SectionLabel label="Metadata" validState={validSteps.metadata} />,
      icon: Code,
      content: <MetadataSetup />,
    },
  ];

  const handleOpenChange = (open: boolean) => {
    setIsSettingsOpen(open);
  };

  return (
    <>
      <Button className="rounded-lg" onClick={() => setIsSettingsOpen(true)}>
        New Key
      </Button>
      <FormProvider {...methods}>
        <form id="new-key-form" onSubmit={methods.handleSubmit(onSubmit)}>
          <NavigableDialog
            isOpen={isSettingsOpen}
            onOpenChange={handleOpenChange}
            title="New Key"
            subTitle="Create a custom API key with your own settings"
            items={settingsNavItems}
            onNavigate={handleSectionNavigation}
            footer={
              <div className="flex justify-center items-center w-full">
                <div className="flex flex-col items-center justify-center w-2/3 gap-2">
                  <Button
                    type="submit"
                    form="new-key-form"
                    variant="primary"
                    size="xlg"
                    className="w-full rounded-lg"
                    disabled={!methods.formState.isValid}
                  >
                    Create new key
                  </Button>
                  <div className="text-xs text-gray-9">
                    This key will be created immediately and ready-to-use right away
                  </div>
                </div>
              </div>
            }
            contentClassName="min-h-[600px]"
            dialogClassName="!min-w-[720px]"
          />
        </form>
      </FormProvider>
    </>
  );
};
