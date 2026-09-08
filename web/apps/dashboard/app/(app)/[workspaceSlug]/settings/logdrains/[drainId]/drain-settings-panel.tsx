"use client";

import {
  Button,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelFooter,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import { FormProvider } from "react-hook-form";
import { DestinationFields, NameField } from "../drain-fields";
import type { DrainSettings } from "./use-drain-settings";

export function DrainSettingsPanel({
  settings,
  isOpen,
  onClose,
}: {
  settings: DrainSettings;
  isOpen: boolean;
  onClose: () => void;
}) {
  const { form, save, update, reset } = settings;

  return (
    <SlidePanel isOpen={isOpen} onClose={onClose} onExitComplete={reset} widthClassName="w-160">
      <SlidePanelHeader className="items-center pb-0">
        <SlidePanelTitle>Settings</SlidePanelTitle>
        <SlidePanelCloseButton />
      </SlidePanelHeader>

      <SlidePanelContent className="flex flex-col">
        <FormProvider {...form}>
          <form onSubmit={save(onClose)} className="flex min-h-0 flex-1 flex-col">
            <div className="flex flex-1 flex-col gap-6 overflow-y-auto px-6 py-5">
              <NameField />
              <DestinationFields tokenRequired={false} />
            </div>
            <SlidePanelFooter className="flex items-center justify-end gap-3">
              <Button type="button" variant="outline" size="md" onClick={onClose}>
                Cancel
              </Button>
              <Button type="submit" variant="primary" size="md" loading={update.isLoading}>
                Save changes
              </Button>
            </SlidePanelFooter>
          </form>
        </FormProvider>
      </SlidePanelContent>
    </SlidePanel>
  );
}
