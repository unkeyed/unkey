"use client";

import {
  Button,
  SlidePanel,
  SlidePanelClose,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelFooter,
  SlidePanelHeader,
  SlidePanelTitle,
} from "@unkey/ui";
import { useState } from "react";
import { FormProvider, useFormState } from "react-hook-form";
import { type ConnectionFormController, useConnectionForm } from "../../hooks/use-connection-form";
import { FORM_MODES } from "../../model/mode";
import type { ListedConnection } from "../../model/target";
import { DeleteConnectionDialog } from "../delete-connection-dialog";
import { ConnectionSteps } from "./connection-steps";

export function ConnectionPanel({
  isOpen,
  ...options
}: Parameters<typeof useConnectionForm>[0] & { isOpen: boolean }) {
  const controller = useConnectionForm(options);
  const { isDirty } = useFormState({ control: controller.form.control });
  const saving = controller.pending !== null;

  return (
    <SlidePanel
      dirty={isDirty && !saving}
      isOpen={isOpen}
      onClose={() => {
        if (!saving) {
          options.onClose();
        }
      }}
    >
      <SlidePanelHeader>
        <SlidePanelTitle>{FORM_MODES[controller.mode].title}</SlidePanelTitle>
        <SlidePanelCloseButton className="mt-0.5" disabled={saving} />
      </SlidePanelHeader>
      <SlidePanelContent>
        <FormProvider {...controller.form}>
          <form
            onSubmit={(event) => event.preventDefault()}
            className="flex h-full flex-col gap-3 overflow-y-auto px-6 pt-1 pb-8"
          >
            <ConnectionSteps controller={controller} />
          </form>
        </FormProvider>
      </SlidePanelContent>
      {controller.before ? (
        <EditFooter connection={controller.before} controller={controller} />
      ) : null}
    </SlidePanel>
  );
}

function EditFooter({
  connection,
  controller,
}: {
  connection: ListedConnection;
  controller: ConnectionFormController;
}) {
  const [isRemoveConfirmOpen, setIsRemoveConfirmOpen] = useState(false);
  const { pending } = controller;

  return (
    <SlidePanelFooter className="flex items-center justify-between gap-3">
      <Button
        type="button"
        variant="outline"
        color="danger"
        size="md"
        loading={pending?.action === "remove"}
        disabled={pending !== null}
        onClick={() => setIsRemoveConfirmOpen(true)}
      >
        Delete connection
      </Button>
      <SlidePanelClose
        disabled={pending !== null}
        render={<Button type="button" variant="outline" size="md" />}
      >
        Close
      </SlidePanelClose>
      <DeleteConnectionDialog
        targetAppName={connection.targetAppName}
        open={isRemoveConfirmOpen}
        onOpenChange={setIsRemoveConfirmOpen}
        onConfirm={() => controller.remove(connection)}
      />
    </SlidePanelFooter>
  );
}
