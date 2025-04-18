"use client";
import { CalendarClock, ChartPie, Code, Gauge, Key2 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import {
  type NavItem,
  NavigableDialog,
} from "@/components/dialog-container/navigable-dialog";
import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { getDefaultValues, processFormData } from "./form-utils";
import { type FormValues, formSchema } from "./schema";
import { GeneralSetup } from "./components/general-setup";

export const CreateKeyDialog = () => {
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

  const settingsNavItems: NavItem[] = [
    {
      id: "general",
      label: "General Setup",
      icon: Key2,
      content: <GeneralSetup />,
    },
    {
      id: "ratelimit",
      label: "Ratelimit",
      icon: Gauge,
      content: <div>Asdsad</div>,
    },
    {
      id: "usage-limit",
      label: "Usage limit",
      icon: ChartPie,
      content: <div>Asdsad</div>,
    },
    {
      id: "expiration",
      label: "Expiration",
      icon: CalendarClock,
      content: <div>Asdsad</div>,
    },
    {
      id: "metadata",
      label: "Metadata",
      icon: Code,
      content: <div>Asdsad</div>,
    },
  ];

  // Handle dialog open/close
  const handleOpenChange = (open: boolean) => {
    if (!open) {
      // Reset form when closing without submission
      methods.reset();
    }
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
            footer={
              <div className="flex justify-center items-center w-full">
                <div className="flex flex-col items-center justify-center w-2/3 gap-2">
                  <Button
                    type="submit"
                    form="new-key-form"
                    variant="primary"
                    size="xlg"
                    className="w-full rounded-lg"
                    disabled={
                      !methods.formState.isValid && methods.formState.isDirty
                    }
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
            dialogClassName="!min-w-[720px]"
          />
        </form>
      </FormProvider>
    </>
  );
};
