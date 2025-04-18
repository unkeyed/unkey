"use client";
import {
  type NavItem,
  NavigableDialog,
} from "@/components/dialog-container/navigable-dialog";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  CalendarClock,
  ChartPie,
  XMark,
  Check,
  Code,
  Gauge,
  Key2,
} from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { GeneralSetup } from "./components/general-setup";
import {
  getDefaultValues,
  getFieldsFromSchema,
  processFormData,
  sectionSchemaMap,
} from "./form-utils";
import { type FormValues, formSchema } from "./schema";
import { RatelimitSetup } from "./components/ratelimit-setup";
import { UsageSetup } from "./components/usage-setup";
import { ExpirationSetup } from "./components/expiration-setup";

type SectionName =
  | "general"
  | "ratelimit"
  | "credits"
  | "expiration"
  | "metadata";

export const CreateKeyDialog = () => {
  const [validSteps, setValidSteps] = useState<
    Record<SectionName, boolean | "initial" | "valid" | "invalid">
  >({
    general: "initial",
    metadata: "initial",
    expiration: "initial",
    ratelimit: "initial",
    credits: "initial",
  });
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
    setIsSettingsOpen(false);
  };

  const handleSectionNavigation = async (fromId: SectionName) => {
    // Skip validation for non-existent sections
    if (!sectionSchemaMap[fromId]) {
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
      label: (
        <div className="w-full justify-between flex items-center">
          General Setup{" "}
          {validSteps.general === "initial" ? null : validSteps.general ===
            "valid" ? (
            <div className="text-success-9 ml-auto">
              <Check className="text-success-9 " size="md-regular" />
            </div>
          ) : (
            <div className="text-success-9 ml-auto">
              <XMark className="text-error-9 " size="md-regular" />
            </div>
          )}
        </div>
      ),
      icon: Key2,
      content: <GeneralSetup />,
    },
    {
      id: "ratelimit",
      label: (
        <div className="w-full justify-between flex items-center">
          Ratelimit
          {validSteps.ratelimit === "initial" ? null : validSteps.ratelimit ===
            "valid" ? (
            <div className="text-success-9 ml-auto">
              <Check className="text-success-9 " size="md-regular" />
            </div>
          ) : (
            <div className="text-success-9 ml-auto">
              <XMark className="text-error-9 " size="md-regular" />
            </div>
          )}
        </div>
      ),
      icon: Gauge,
      content: <RatelimitSetup />,
    },
    {
      id: "credits",
      label: (
        <div className="w-full justify-between flex items-center">
          Credits
          {validSteps.credits === "initial" ? null : validSteps.credits ===
            "valid" ? (
            <div className="text-success-9 ml-auto">
              <Check className="text-success-9 " size="md-regular" />
            </div>
          ) : (
            <div className="text-success-9 ml-auto">
              <XMark className="text-error-9 " size="md-regular" />
            </div>
          )}
        </div>
      ),
      icon: ChartPie,
      content: <UsageSetup />,
    },
    {
      id: "expiration",
      label: "Expiration",
      icon: CalendarClock,
      content: <ExpirationSetup />,
    },
    {
      id: "metadata",
      label: "Metadata",
      icon: Code,
      content: <div>Metadata Component</div>,
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
                    This key will be created immediately and ready-to-use right
                    away
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
