"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { useProjectData } from "../../../data-provider";
import { defaultConnectionName } from "../../connection-rules";
import type { AfterSave } from "../../hooks/use-connection-actions";
import { type ConnectionsData, useTargetDeployments } from "../../hooks/use-connections-data";
import {
  type ConnectionFormValues,
  connectionFormSchema,
  defaultEnvs,
  previewTargetFor,
} from "../../model/form";
import { ADD_STEPS, completeStep } from "../../model/steps";
import { AppPicker } from "./app-picker";
import { ConnectionForm } from "./connection-form";
import { AppName, ConnectionSteps } from "./connection-steps";

export function AddConnectionPanel({
  data,
  isOpen,
  onClose,
  onSave,
}: {
  data: ConnectionsData;
  isOpen: boolean;
  onClose: () => void;
  onSave: (values: ConnectionFormValues, after: AfterSave) => Promise<boolean>;
}) {
  const { envs, merged, targets, requiresExplicitPreviewTarget } = data;
  const { apps: projectApps } = useProjectData();
  const form = useForm<ConnectionFormValues>({
    resolver: zodResolver(
      connectionFormSchema({ requiresExplicitPreviewTarget, nameRequired: false }),
    ),
    defaultValues: { targetAppId: "", name: "", envs: defaultEnvs(null, envs) },
  });
  const [steps, setSteps] = useState(ADD_STEPS);
  const [committing, setCommitting] = useState<AfterSave | null>(null);
  const targetAppId = useWatch({ control: form.control, name: "targetAppId" });
  const page = useTargetDeployments({
    projectId: data.projectId,
    appId: data.appId,
    envs,
    targetAppId,
  });
  const connected = new Set(merged.map((connection) => connection.targetAppId));
  const target = targets.apps.find((app) => app.id === targetAppId);
  const takenNames = new Set(
    merged.flatMap((connection) => Object.values(connection.byEnv).map((row) => row.name)),
  );

  const pickApp = (id: string) => {
    form.setValue("targetAppId", id, { shouldDirty: true, shouldValidate: true });
    const picked = targets.apps.find((app) => app.id === id);
    if (picked && !form.getFieldState("name").isDirty) {
      form.setValue(
        "name",
        defaultConnectionName(picked.slug, (name) => takenNames.has(name)) ?? "",
      );
    }
    form.setValue(
      "envs",
      defaultEnvs(
        previewTargetFor({
          requiresExplicitPreviewTarget,
          environments: targets.environments,
          targetAppId: id,
        }),
        envs,
      ),
      { shouldDirty: true },
    );
    setSteps(completeStep);
  };

  const commit = (after: AfterSave) =>
    form.handleSubmit(async (values) => {
      setCommitting(after);
      const saved = await onSave(values, after);
      setCommitting(null);
      if (saved) {
        form.reset(values);
        onClose();
      }
    })();

  return (
    <ConnectionForm title="Add connection" isOpen={isOpen} onClose={onClose} form={form}>
      <ConnectionSteps
        mode="add"
        steps={steps}
        onStepsChange={setSteps}
        onCommit={commit}
        committing={committing}
        app={{
          locked: false,
          summary: target ? <AppName name={target.name} /> : null,
          picker: (
            <AppPicker
              apps={targets.apps}
              projectApps={projectApps}
              connectedIds={connected}
              onPick={pickApp}
            />
          ),
        }}
        target={target}
        envs={envs}
        targets={targets}
        page={page}
        requiresExplicitPreviewTarget={requiresExplicitPreviewTarget}
      />
    </ConnectionForm>
  );
}
