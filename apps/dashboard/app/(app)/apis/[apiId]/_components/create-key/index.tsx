"use client";
import {
  NavigableDialogBody,
  NavigableDialogContent,
  NavigableDialogFooter,
  NavigableDialogHeader,
  NavigableDialogNav,
  NavigableDialogRoot,
} from "@/components/dialog-container/navigable-dialog";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import { zodResolver } from "@hookform/resolvers/zod";
import type { IconProps } from "@unkey/icons/src/props";
import { Button } from "@unkey/ui";
import { type FC, useState } from "react";
import { FormProvider } from "react-hook-form";
import { toast } from "sonner";
import { KeyCreatedSuccessDialog } from "./components/key-created-success-dialog";
import { SectionLabel } from "./components/section-label";
import { type DialogSectionName, SECTIONS } from "./constants";
import { getDefaultValues, processFormData } from "./form-utils";
import { useCreateKey } from "./hooks/use-create-key";
import { useValidateSteps } from "./hooks/use-validate-steps";
import { type FormValues, formSchema } from "./schema";

// Storage key for saving form state
const FORM_STORAGE_KEY = "unkey_create_key_form_state";

export const CreateKeyDialog = ({
  keyspaceId,
  apiId,
}: {
  keyspaceId: string | null;
  apiId: string;
}) => {
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [successDialogOpen, setSuccessDialogOpen] = useState(false);
  const [createdKeyData, setCreatedKeyData] = useState<{
    key: string;
    id: string;
    name?: string;
  } | null>(null);

  const methods = usePersistedForm<FormValues>(
    FORM_STORAGE_KEY,
    {
      resolver: zodResolver(formSchema),
      mode: "onChange",

      shouldFocusError: true,
      shouldUnregister: true,
      defaultValues: getDefaultValues(),
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
  });

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      saveCurrentValues();
    }
    setIsSettingsOpen(open);
  };

  const onSubmit = async (data: FormValues) => {
    const finalData = processFormData(data);
    if (!keyspaceId) {
      toast.error("Failed to Create Key", {
        description: "An unexpected error occurred. Please try again later.",
        action: {
          label: "Contact Support",
          onClick: () => window.open("https://support.unkey.dev", "_blank"),
        },
      });
      return;
    }

    try {
      await key.mutateAsync({
        keyAuthId: keyspaceId,
        ...finalData,
      });
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
              <NavigableDialogNav
                items={SECTIONS.map((section) => ({
                  id: section.id,
                  label: <SectionLabel label={section.label} validState={validSteps[section.id]} />,
                  icon: section.icon as FC<IconProps>,
                }))}
                onNavigate={handleSectionNavigation}
              />
              <NavigableDialogContent
                items={SECTIONS.map((section) => ({
                  id: section.id,
                  content: section.content(),
                }))}
                className="min-h-[600px]"
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
      <KeyCreatedSuccessDialog
        apiId={apiId}
        keyspaceId={keyspaceId}
        isOpen={successDialogOpen}
        onClose={handleSuccessDialogClose}
        keyData={createdKeyData}
        onCreateAnother={openNewKeyDialog}
      />
    </>
  );
};
