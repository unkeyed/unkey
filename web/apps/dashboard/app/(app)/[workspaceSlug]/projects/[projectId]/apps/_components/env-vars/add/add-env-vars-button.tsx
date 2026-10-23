import { IconPlusOutline18 } from "@unkey/icons";
import {
  Button,
  SlidePanel,
  SlidePanelClose,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelDescription,
  SlidePanelFooter,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import { useState } from "react";
import { AddEnvVarsFields, useAddEnvVarsForm } from "./add-env-vars-fields";

export function AddEnvVarsButton() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <Button size="md" variant="primary" onClick={() => setIsOpen(true)}>
        <IconPlusOutline18 />
        Add environment variable
      </Button>
      <AddEnvVarsPanel isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </>
  );
}

type PanelMode = "add" | "manage";

const panelCopy: Record<PanelMode, { title: string; description: string | null }> = {
  add: {
    title: "Add environment variables",
    description: "Paste or drop a .env file into the table to add many at once.",
  },
  manage: {
    title: "Environment variables",
    description: null,
  },
};

export function AddEnvVarsPanel({
  isOpen,
  onClose,
  onSaved,
  mode = "add",
}: {
  isOpen: boolean;
  onClose: () => void;
  onSaved?: () => void;
  mode?: PanelMode;
}) {
  const form = useAddEnvVarsForm({
    showSaved: mode === "manage",
    onSaved: () => {
      onClose();
      onSaved?.();
    },
  });
  const resetAndClose = () => {
    if (form.isPending) {
      return;
    }
    form.reset();
    onClose();
  };

  return (
    <SlidePanel dirty={form.isDirty} isOpen={isOpen} onClose={resetAndClose} widthClassName="w-240">
      <SlidePanelHeader>
        <div className="flex flex-col gap-0.5">
          <SlidePanelTitle>{panelCopy[mode].title}</SlidePanelTitle>
          {panelCopy[mode].description ? (
            <SlidePanelDescription>{panelCopy[mode].description}</SlidePanelDescription>
          ) : null}
        </div>
        <SlidePanelCloseButton className="mt-0.5" />
      </SlidePanelHeader>

      <SlidePanelContent>
        <form onSubmit={form.onSubmit} className="flex h-full flex-col">
          <div className="flex flex-1 flex-col gap-6 overflow-y-auto px-6 pt-4 pb-6">
            <AddEnvVarsFields form={form} />
          </div>
          <SlidePanelFooter className="flex items-center justify-end gap-3 bg-raised py-5">
            <SlidePanelClose
              disabled={form.isPending}
              render={<Button type="button" variant="outline" size="md" />}
            >
              Cancel
            </SlidePanelClose>
            <Button
              type="submit"
              variant="primary"
              size="md"
              className="px-3"
              loading={form.isPending}
              disabled={form.isPending || !form.canSubmit || !form.isDirty}
            >
              Save
            </Button>
          </SlidePanelFooter>
        </form>
      </SlidePanelContent>
    </SlidePanel>
  );
}
