import { useEffect, useState } from "react";
import type { FieldPath, UseFormTrigger } from "react-hook-form";
import { DEFAULT_STEP_STATES, type DialogSectionName, SECTIONS } from "../create-key.constants";
import type { FormValues } from "../create-key.schema";
import { getFieldsFromSchema, isFeatureEnabled, sectionSchemaMap } from "../create-key.utils";
import type { SectionName, SectionState } from "../types";

// Custom hook to handle form validation on dialog open
export const useValidateSteps = (
  isSettingsOpen: boolean,
  loadSavedValues: () => Promise<boolean>,
  trigger: UseFormTrigger<FormValues>,
  getValues: () => FormValues,
) => {
  const [validSteps, setValidSteps] =
    useState<Record<DialogSectionName, SectionState>>(DEFAULT_STEP_STATES);

  useEffect(() => {
    if (isSettingsOpen) {
      const loadAndValidate = async () => {
        const loaded = await loadSavedValues();
        if (loaded) {
          // Validate all sections after loading
          const newValidSteps = { ...DEFAULT_STEP_STATES };
          for (const section of SECTIONS) {
            // Skip validating non-existent sections
            if (!sectionSchemaMap[section.id as SectionName]) {
              continue;
            }
            // Skip validation if the feature is not enabled
            if (
              section.id !== "general" &&
              !isFeatureEnabled(section.id as SectionName, getValues())
            ) {
              newValidSteps[section.id] = "initial";
              continue;
            }
            // Get fields from the schema and validate
            const schema = sectionSchemaMap[section.id as SectionName];
            const fieldsToValidate = getFieldsFromSchema(schema);
            if (fieldsToValidate.length > 0) {
              const result = await trigger(fieldsToValidate as FieldPath<FormValues>[]);
              newValidSteps[section.id] = result ? "valid" : "invalid";
            }
          }
          setValidSteps(newValidSteps);
        }
      };
      loadAndValidate();
    }
  }, [isSettingsOpen, loadSavedValues, trigger, getValues]);

  // Function to validate a specific section
  const validateSection = async (sectionId: DialogSectionName) => {
    // Skip validation for non-existent sections
    if (!sectionSchemaMap[sectionId as SectionName]) {
      return true;
    }

    // Skip validation if the feature is not enabled
    if (sectionId !== "general" && !isFeatureEnabled(sectionId as SectionName, getValues())) {
      setValidSteps((prevState) => ({
        ...prevState,
        [sectionId]: "initial",
      }));
      return true;
    }

    // Get the schema for the section
    const schema = sectionSchemaMap[sectionId as SectionName];
    // Get fields from the schema
    const fieldsToValidate = getFieldsFromSchema(schema);
    // Skip validation if no fields to validate
    if (fieldsToValidate.length === 0) {
      return true;
    }
    // Trigger validation for the fields
    const result = await trigger(fieldsToValidate as FieldPath<FormValues>[]);
    setValidSteps((prevState) => ({
      ...prevState,
      [sectionId]: result ? "valid" : "invalid",
    }));

    return result;
  };

  // Function to reset validation states to default
  const resetValidSteps = () => {
    setValidSteps(DEFAULT_STEP_STATES);
  };

  return { validSteps, validateSection, resetValidSteps };
};
