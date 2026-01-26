"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { Navbar } from "@/components/navigation/navbar";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "@unkey/icons";
import {
  Button,
  Loading,
  NavigableDialogBody,
  NavigableDialogContent,
  NavigableDialogFooter,
  NavigableDialogHeader,
  NavigableDialogNav,
  NavigableDialogRoot,
  toast,
} from "@unkey/ui";
import { Suspense, useEffect, useState } from "react";
import { FormProvider } from "react-hook-form";
import { KeyCreatedSuccessDialog } from "./components/key-created-success-dialog";
import { SectionLabel } from "./components/section-label";
import { type DialogSectionName, SECTIONS } from "./create-key.constants";
import { type FormValues, formSchema } from "./create-key.schema";
import { formValuesToApiInput, getDefaultValues } from "./create-key.utils";
import { useCreateKey } from "./hooks/use-create-key";
import { useValidateSteps } from "./hooks/use-validate-steps";

// Storage key for saving form state
const FORM_STORAGE_KEY = "unkey_create_key_form_state";

export const CreateKeyDialog = ({
  keyspaceId,
  apiId,
  copyIdValue,
  keyspaceDefaults,
}: {
  keyspaceId: string | null;
  apiId: string;
  copyIdValue?: string;
  keyspaceDefaults: {
    prefix?: string;
    bytes?: number;
  } | null;
}) => {
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [successDialogOpen, setSuccessDialogOpen] = useState(false);
  const [createdKeyData, setCreatedKeyData] = useState<{
    key: string;
    id: string;
    name?: string;
  } | null>(null);
  const [dialogKey, setDialogKey] = useState(0);

  const methods = usePersistedForm<FormValues>(
    FORM_STORAGE_KEY,
    {
      resolver: zodResolver(formSchema),
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
      defaultValues: getDefaultValues(keyspaceDefaults),
    },
    "memory",
  );

  const {
    handleSubmit,
    formState,
    getValues,
    reset,
    trigger,
    clearPersistedData,
    loadSavedValues,
    saveCurrentValues,
  } = methods;

  // Update form defaults when keyspace defaults change after revalidation
  useEffect(() => {
    const newDefaults = getDefaultValues(keyspaceDefaults);
    clearPersistedData();
    reset(newDefaults);
  }, [keyspaceDefaults, reset, clearPersistedData]);

  const { validSteps, validateSection, resetValidSteps } = useValidateSteps(
    isSettingsOpen,
    loadSavedValues,
    trigger,
    getValues,
  );

  const key = useCreateKey((data) => {
    if (data?.key && data?.keyId) {
      setCreatedKeyData({
        key: data.key,
        id: data.keyId,
        name: data.name,
      });
      setSuccessDialogOpen(true);
    }

    // Clean up form state
    clearPersistedData();
    reset(getDefaultValues());
    setIsSettingsOpen(false);
    resetValidSteps();
    // Force dialog to remount and reset to initial state (general section)
    setDialogKey((prev) => prev + 1);
  });

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      saveCurrentValues();
      // Reset to general step when closing, so next time it opens on the first step
      setDialogKey((prev) => prev + 1);
    }
    setIsSettingsOpen(open);
  };

  const onSubmit = async (data: FormValues) => {
    if (!keyspaceId) {
      toast.error("Failed to Create Key", {
        description: "An unexpected error occurred. Please try again later.",
        action: {
          label: "Contact Support",
          onClick: () => window.open("mailto:support@unkey.dev", "_blank"),
        },
      });
      return;
    }
    const finalData = formValuesToApiInput(data, keyspaceId);

    try {
      await key.mutateAsync(finalData);
    } catch {
      // `useCreateKey` already shows a toast, but we still need to
      // prevent unhandled‐rejection noise in the console.
    }
  };

  const handleSectionNavigation = async (fromId: DialogSectionName) => {
    await validateSection(fromId);
    return true;
  };

  const handleSuccessDialogClose = () => {
    setSuccessDialogOpen(false);
    setCreatedKeyData(null);
  };

  const openNewKeyDialog = () => {
    setIsSettingsOpen(true);
  };

  return (
    <>
      <Navbar.Actions>
        <NavbarActionButton title="Create new key" onClick={() => setIsSettingsOpen(true)}>
          <Plus />
          Create new key
        </NavbarActionButton>
        <CopyableIDButton value={copyIdValue ?? apiId} />
      </Navbar.Actions>

      <FormProvider {...methods}>
        <form id="new-key-form" onSubmit={handleSubmit(onSubmit)}>
          <NavigableDialogRoot
            key={dialogKey}
            isOpen={isSettingsOpen}
            onOpenChange={handleOpenChange}
            dialogClassName="w-[90%] md:w-[70%] lg:w-[70%] xl:w-[50%] 2xl:w-[45%] max-w-[940px] max-h-[90vh] sm:max-h-[90vh] md:max-h-[70vh] lg:max-h-[90vh] xl:max-h-[80vh]"
          >
            <NavigableDialogHeader
              title="New Key"
              subTitle="Create a custom API key with your own settings"
            />
            <NavigableDialogBody>
              <NavigableDialogNav
                items={SECTIONS.map((section) => ({
                  id: section.id,
                  label: <SectionLabel label={section.label} validState={validSteps[section.id]} />,
                  icon: section.icon,
                }))}
                onNavigate={handleSectionNavigation}
                initialSelectedId="general"
              />
              <NavigableDialogContent
                items={SECTIONS.map((section) => ({
                  id: section.id,
                  content: section.content(),
                }))}
              />
            </NavigableDialogBody>
            <NavigableDialogFooter>
              <div className="flex justify-center items-center w-full">
                <div className="flex flex-col items-center justify-center w-2/3 gap-2">
                  <Button
                    type="submit"
                    form="new-key-form"
                    variant="primary"
                    size="xlg"
                    className="w-full rounded-lg"
                    disabled={!formState.isValid}
                    loading={key.isLoading}
                  >
                    Create new key
                  </Button>
                  <div className="text-xs text-gray-9">
                    This key will be created immediately and ready-to-use right away
                  </div>
                </div>
              </div>
            </NavigableDialogFooter>
          </NavigableDialogRoot>
        </form>
      </FormProvider>
      {/* Success Dialog */}
      <Suspense fallback={<Loading type="spinner" />}>
        <KeyCreatedSuccessDialog
          apiId={apiId}
          keyspaceId={keyspaceId}
          isOpen={successDialogOpen}
          onClose={handleSuccessDialogClose}
          keyData={createdKeyData}
          onCreateAnother={openNewKeyDialog}
        />
      </Suspense>
    </>
  );
};
