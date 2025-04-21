"use client";
import {
  NavigableDialogBody,
  NavigableDialogContent,
  NavigableDialogFooter,
  NavigableDialogHeader,
  NavigableDialogNav,
  NavigableDialogRoot,
} from "@/components/dialog-container/navigable-dialog";
import { zodResolver } from "@hookform/resolvers/zod";
import type { IconProps } from "@unkey/icons/src/props";
import { Button } from "@unkey/ui";
import { type FC, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { SectionLabel } from "./components/section-label";
import { DEFAULT_STEP_STATES, type DialogSectionName, SECTIONS } from "./constants";
import {
  getDefaultValues,
  getFieldsFromSchema,
  isFeatureEnabled,
  processFormData,
  sectionSchemaMap,
} from "./form-utils";
import { type FormValues, formSchema } from "./schema";
import type { SectionName, SectionState } from "./types";

export const CreateKeyDialog = () => {
  const [validSteps, setValidSteps] =
    useState<Record<DialogSectionName, SectionState>>(DEFAULT_STEP_STATES);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);

  const methods = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
    shouldFocusError: true,
    shouldUnregister: true,
    defaultValues: getDefaultValues(),
  });

  const { handleSubmit, formState, getValues, reset, trigger } = methods;

  const onSubmit = (data: FormValues) => {
    processFormData(data);
    reset(getDefaultValues());
    setIsSettingsOpen(false);
    setValidSteps(DEFAULT_STEP_STATES);
  };

  const handleSectionNavigation = async (fromId: DialogSectionName) => {
    // Skip validation for non-existent sections
    if (!sectionSchemaMap[fromId as SectionName]) {
      return true;
    }

    // Skip validation if the feature is not enabled
    if (fromId !== "general" && !isFeatureEnabled(fromId as SectionName, getValues())) {
      setValidSteps((prevState) => ({
        ...prevState,
        [fromId]: "initial",
      }));
      return true;
    }

    // Get the schema for the section
    const schema = sectionSchemaMap[fromId as SectionName];
    // Get fields from the schema
    const fieldsToValidate = getFieldsFromSchema(schema);
    // Skip validation if no fields to validate
    if (fieldsToValidate.length === 0) {
      return true;
    }

    // Trigger validation for the fields
    const result = await trigger(fieldsToValidate as any);
    setValidSteps((prevState) => ({
      ...prevState,
      [fromId]: result ? "valid" : "invalid",
    }));
    // Always allow navigation
    return true;
  };

  // Convert sections to nav items with section labels
  const navItems = SECTIONS.map((section) => ({
    id: section.id,
    label: <SectionLabel label={section.label} validState={validSteps[section.id]} />,
    icon: section.icon as FC<IconProps>,
  }));

  // Convert sections to content items
  const contentItems = SECTIONS.map((section) => ({
    id: section.id,
    content: section.content(),
  }));

  const handleOpenChange = (open: boolean) => {
    setIsSettingsOpen(open);
  };

  const footerContent = (
    <div className="flex justify-center items-center w-full">
      <div className="flex flex-col items-center justify-center w-2/3 gap-2">
        <Button
          type="submit"
          form="new-key-form"
          variant="primary"
          size="xlg"
          className="w-full rounded-lg"
          disabled={!formState.isValid}
        >
          Create new key
        </Button>
        <div className="text-xs text-gray-9">
          This key will be created immediately and ready-to-use right away
        </div>
      </div>
    </div>
  );

  return (
    <>
      <Button className="rounded-lg" onClick={() => setIsSettingsOpen(true)}>
        New Key
      </Button>
      <FormProvider {...methods}>
        <form id="new-key-form" onSubmit={handleSubmit(onSubmit)}>
          <NavigableDialogRoot
            isOpen={isSettingsOpen}
            onOpenChange={handleOpenChange}
            dialogClassName="!min-w-[760px]"
          >
            <NavigableDialogHeader
              title="New Key"
              subTitle="Create a custom API key with your own settings"
            />
            <NavigableDialogBody>
              <NavigableDialogNav items={navItems} onNavigate={handleSectionNavigation} />
              <NavigableDialogContent items={contentItems} className="min-h-[600px]" />
            </NavigableDialogBody>
            <NavigableDialogFooter>{footerContent}</NavigableDialogFooter>
          </NavigableDialogRoot>
        </form>
      </FormProvider>
    </>
  );
};
