"use client";
import { DialogContainer } from "@/components/dialog-container";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { FormProvider } from "react-hook-form";
import { type FormValues, rbacRoleSchema } from "./upsert-role.schema";

// Storage key for saving form state
const FORM_STORAGE_KEY = "unkey_upsert_role_form_state";

export const UpsertRoleDialog = () => {
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const methods = usePersistedForm<FormValues>(
    FORM_STORAGE_KEY,
    {
      resolver: zodResolver(rbacRoleSchema),
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
    },
    "memory",
  );

  // const {
  //   handleSubmit,
  //   formState,
  //   getValues,
  //   reset,
  //   trigger,
  //   clearPersistedData,
  //   loadSavedValues,
  //   saveCurrentValues,
  // } = methods;
  //
  // // Update form defaults when keyspace defaults change after revalidation
  // useEffect(() => {
  //   const newDefaults = getDefaultValues(keyspaceDefaults);
  //   clearPersistedData();
  //   reset(newDefaults);
  // }, [keyspaceDefaults, reset, clearPersistedData]);
  //
  // const key = useCreateKey((data) => {
  //   if (data?.key && data?.keyId) {
  //     setCreatedKeyData({
  //       key: data.key,
  //       id: data.keyId,
  //       name: data.name,
  //     });
  //     setSuccessDialogOpen(true);
  //   }
  //
  //   // Clean up form state
  //   clearPersistedData();
  //   reset(getDefaultValues());
  //   setIsSettingsOpen(false);
  //   resetValidSteps();
  // });

  //
  // const onSubmit = async (data: FormValues) => {
  //   if (!keyspaceId) {
  //     toast.error("Failed to Create Key", {
  //       description: "An unexpected error occurred. Please try again later.",
  //       action: {
  //         label: "Contact Support",
  //         onClick: () => window.open("https://support.unkey.dev", "_blank"),
  //       },
  //     });
  //     return;
  //   }
  //   const finalData = formValuesToApiInput(data, keyspaceId);
  //
  //   try {
  //     await key.mutateAsync(finalData);
  //   } catch {
  //     // `useCreateKey` already shows a toast, but we still need to
  //     // prevent unhandled‐rejection noise in the console.
  //   }
  // };
  //

  return (
    <>
      <Navbar.Actions>
        <NavbarActionButton title="Create new role" onClick={() => setIsDialogOpen(true)}>
          <Plus />
          Create new key
        </NavbarActionButton>
      </Navbar.Actions>
      <FormProvider {...methods}>
        <form id="new-role-form">
          <DialogContainer
            title="Create new role"
            subTitle="Define a role and assign permissions"
            isOpen={isDialogOpen}
            onOpenChange={setIsDialogOpen}
            contentClassName="w-[90%] md:w-[70%] lg:w-[70%] xl:w-[50%] 2xl:w-[45%] max-w-[940px] max-h-[90vh] sm:max-h-[90vh] md:max-h-[70vh] lg:max-h-[90vh] xl:max-h-[80vh]"
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="submit"
                  form="create-namespace-form"
                  variant="primary"
                  size="xlg"
                  className="w-full rounded-lg"
                >
                  Create new role
                </Button>
                <div className="text-gray-9 text-xs">
                  Namespaces can be used to separate different rate limiting concerns
                </div>
              </div>
            }
          />
        </form>
      </FormProvider>
    </>
  );
};
