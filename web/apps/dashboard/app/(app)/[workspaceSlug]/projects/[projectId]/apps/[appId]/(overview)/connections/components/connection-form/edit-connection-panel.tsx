"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
  SlidePanelClose,
} from "@unkey/ui";
import { useState } from "react";
import { useForm } from "react-hook-form";
import type { AfterSave } from "../../hooks/use-connection-actions";
import { type ConnectionsData, useTargetDeployments } from "../../hooks/use-connections-data";
import {
  type ConnectionFormValues,
  connectionFormSchema,
  defaultEnvs,
  formValuesOf,
  previewTargetFor,
} from "../../model/form";
import type { MergedConnection } from "../../model/merge";
import { EDIT_STEPS } from "../../model/steps";
import { ConnectionForm } from "./connection-form";
import { AppName, ConnectionSteps } from "./connection-steps";

export function EditConnectionPanel({
  data,
  row,
  isOpen,
  onClose,
  onSave,
  onRemove,
}: {
  data: ConnectionsData;
  row: MergedConnection | undefined;
  isOpen: boolean;
  onClose: () => void;
  onSave: (
    before: MergedConnection,
    values: ConnectionFormValues,
    after: AfterSave,
  ) => Promise<boolean>;
  onRemove: (connection: MergedConnection) => Promise<boolean>;
}) {
  const { envs, targets, requiresExplicitPreviewTarget } = data;
  const [rowAtOpen] = useState(row);
  const [steps, setSteps] = useState(EDIT_STEPS);
  const [committing, setCommitting] = useState<AfterSave | null>(null);
  const [isRemoveConfirmOpen, setIsRemoveConfirmOpen] = useState(false);
  const form = useForm<ConnectionFormValues>({
    resolver: zodResolver(
      connectionFormSchema({ requiresExplicitPreviewTarget, nameRequired: true }),
    ),
    defaultValues: rowAtOpen
      ? formValuesOf(
          rowAtOpen,
          defaultEnvs(
            previewTargetFor({
              requiresExplicitPreviewTarget,
              environments: targets.environments,
              targetAppId: rowAtOpen.targetAppId,
            }),
            envs,
          ),
        )
      : undefined,
  });
  const page = useTargetDeployments({
    projectId: data.projectId,
    appId: data.appId,
    envs,
    targetAppId: rowAtOpen?.targetAppId ?? "",
  });

  if (!rowAtOpen) {
    return null;
  }

  const commit = (after: AfterSave) =>
    form.handleSubmit(async (values) => {
      setCommitting(after);
      const saved = await onSave(rowAtOpen, values, after);
      setCommitting(null);
      if (saved) {
        form.reset(values);
        onClose();
      }
    })();

  return (
    <ConnectionForm
      title="Edit connection"
      isOpen={isOpen}
      onClose={onClose}
      form={form}
      footer={
        <>
          <Button
            type="button"
            variant="outline"
            color="danger"
            size="md"
            onClick={() => setIsRemoveConfirmOpen(true)}
          >
            Delete connection
          </Button>
          <SlidePanelClose render={<Button type="button" variant="outline" size="md" />}>
            Close
          </SlidePanelClose>
          <AlertDialog open={isRemoveConfirmOpen} onOpenChange={setIsRemoveConfirmOpen}>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>
                  Delete the {rowAtOpen.targetAppName} connection?
                </AlertDialogTitle>
                <AlertDialogDescription>
                  New deployments will not include this connection. Existing deployments keep their
                  saved connection.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  color="danger"
                  onClick={async () => {
                    if (await onRemove(rowAtOpen)) {
                      form.reset();
                      onClose();
                    }
                  }}
                >
                  Delete connection
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </>
      }
    >
      <ConnectionSteps
        mode="edit"
        steps={steps}
        onStepsChange={setSteps}
        onCommit={commit}
        committing={committing}
        app={{ locked: true, summary: <AppName name={rowAtOpen.targetAppName} />, picker: null }}
        target={{ id: rowAtOpen.targetAppId, name: rowAtOpen.targetAppName }}
        savedName={rowAtOpen.name}
        envs={envs}
        targets={targets}
        page={page}
        requiresExplicitPreviewTarget={requiresExplicitPreviewTarget}
      />
    </ConnectionForm>
  );
}
