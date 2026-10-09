import { IconPlusOutline18 } from "@unkey/icons";
import {
  Button,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelDescription,
  SlidePanelFooter,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import { useState } from "react";
import {
  AddEnvVarsFields,
  useAddEnvVarsForm,
} from "../../../../../_components/env-vars/add/add-env-vars-fields";

export function AddEnvVarsButton() {
  const [isOpen, setIsOpen] = useState(false);
  const close = () => setIsOpen(false);
  const form = useAddEnvVarsForm({ onAdded: close });
  const resetAndClose = () => {
    form.reset();
    close();
  };

  return (
    <>
      <Button size="md" variant="primary" onClick={() => setIsOpen(true)}>
        <IconPlusOutline18 />
        Add environment variable
      </Button>
      <SlidePanel
        dirty={form.isDirty}
        isOpen={isOpen}
        onClose={resetAndClose}
        widthClassName="w-240"
      >
        <SlidePanelHeader>
          <div className="flex flex-col gap-0.5">
            <SlidePanelTitle>Add environment variables</SlidePanelTitle>
            <SlidePanelDescription>
              Paste or drop a .env file into the table to add many at once.
            </SlidePanelDescription>
          </div>
          <SlidePanelCloseButton className="mt-0.5" />
        </SlidePanelHeader>

        <SlidePanelContent>
          <form onSubmit={form.onSubmit} className="flex h-full flex-col">
            <div className="flex flex-1 flex-col gap-6 overflow-y-auto px-6 pt-4 pb-6">
              <AddEnvVarsFields form={form} />
            </div>
            <SlidePanelFooter className="flex items-center justify-end gap-4 bg-raised py-5">
              <Button
                type="submit"
                variant="primary"
                size="md"
                className="px-3"
                loading={form.isPending}
                disabled={form.isPending || form.targetEnvironmentIds.length === 0}
              >
                Save
              </Button>
            </SlidePanelFooter>
          </form>
        </SlidePanelContent>
      </SlidePanel>
    </>
  );
}
