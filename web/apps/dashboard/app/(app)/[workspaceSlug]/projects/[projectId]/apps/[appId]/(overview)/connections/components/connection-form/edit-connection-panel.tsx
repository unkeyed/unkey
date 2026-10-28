"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
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
}: {
  data: ConnectionsData;
  row: MergedConnection | undefined;
  isOpen: boolean;
  onClose: () => void;
  onSave: (before: MergedConnection, values: ConnectionFormValues) => Promise<boolean>;
}) {
  const { envs, targets, requiresExplicitPreviewTarget } = data;
  const [rowAtOpen] = useState(row);
  const [steps, setSteps] = useState(EDIT_STEPS);
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

  const onSubmit = async (values: ConnectionFormValues) => {
    if (await onSave(rowAtOpen, values)) {
      onClose();
    }
  };

  return (
    <ConnectionForm
      title="Edit connection"
      isOpen={isOpen}
      onClose={onClose}
      form={form}
      onSubmit={onSubmit}
    >
      <ConnectionSteps
        mode="edit"
        steps={steps}
        onStepsChange={setSteps}
        created={false}
        onDone={onClose}
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
