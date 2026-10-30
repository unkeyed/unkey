"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import {
  type ConnectionFormValues,
  connectionFormSchema,
  formValuesOf,
  planSave,
} from "../model/form";
import { FORM_MODES, type FormMode } from "../model/mode";
import { connectionHostVariable, defaultConnectionName, takenNamesOf } from "../model/name";
import { firstStepWithError, openStep, pickedApp } from "../model/steps";
import type { ListedConnection } from "../model/target";
import type { TargetChoices } from "../model/target-options";
import { type ConnectionMutation, removalOf } from "./use-connection-actions";
import { type ConnectionsSnapshot, useTargetDeployments } from "./use-connections-data";

export function useConnectionForm({
  projectId,
  appId,
  environmentId,
  snapshot,
  before,
  mutation,
  onClose,
}: {
  projectId: string;
  appId: string;
  environmentId: string;
  snapshot: ConnectionsSnapshot;
  before: ListedConnection | null;
  mutation: ConnectionMutation;
  onClose: () => void;
}) {
  const { connections, targets } = snapshot;
  const mode: FormMode = before ? "edit" : "add";
  const form = useForm<ConnectionFormValues>({
    resolver: zodResolver(connectionFormSchema(connections)),
    defaultValues: before
      ? formValuesOf(before)
      : { targetAppId: "", name: "", target: { targetType: "automatic" } },
  });
  const [steps, setSteps] = useState(FORM_MODES[mode].initialSteps);
  const targetAppId = useWatch({ control: form.control, name: "targetAppId" });
  const targetType = useWatch({ control: form.control, name: "target.targetType" });
  const deployments = useTargetDeployments({
    projectId,
    appId,
    environmentId,
    targetAppId,
    known: targets.deployments,
    enabled: targetType === "deployment",
  });
  const target = before
    ? { id: before.targetAppId, name: before.targetAppName }
    : targets.apps.find((app) => app.id === targetAppId);
  const choices: TargetChoices | null = target
    ? {
        targetAppId: target.id,
        targetName: target.name,
        environments: targets.environments,
        deployments: deployments.choices,
      }
    : null;
  const pickApp = (id: string) => {
    const appChanged = id !== form.getValues("targetAppId");
    setSteps((state) => pickedApp(state, appChanged));
    if (!appChanged) {
      return;
    }
    form.setValue("targetAppId", id, { shouldDirty: true, shouldValidate: true });
    const picked = targets.apps.find((app) => app.id === id);
    if (picked && !form.getFieldState("name").isDirty) {
      const taken = takenNamesOf(connections, id);
      form.setValue("name", defaultConnectionName(picked.slug, (name) => taken.has(name)) ?? "");
    }
    form.setValue("target", { targetType: "automatic" }, { shouldDirty: true });
  };

  const finish = (values?: ConnectionFormValues) => {
    form.reset(values);
    onClose();
  };

  const commit = () =>
    form.handleSubmit(
      (values) => {
        const op = planSave(before, values);
        if (!op) {
          finish(values);
          return;
        }
        mutation.run(
          {
            action: mode,
            op,
            description:
              op.type === "update" && op.name !== undefined
                ? `Update your code to use ${connectionHostVariable(op.name)}.`
                : undefined,
          },
          () => finish(values),
        );
      },
      (errors) => {
        const step = firstStepWithError(errors);
        if (step) {
          setSteps((state) => openStep(state, step));
        }
      },
    )();

  const remove = (connection: ListedConnection) => mutation.run(removalOf(connection), finish);

  return {
    mode,
    before,
    form,
    steps,
    setSteps,
    target,
    choices,
    deployments: deployments.list,
    apps: targets.apps,
    connectedIds: new Set(connections.map((connection) => connection.targetAppId)),
    pending: mutation.pending,
    pickApp,
    commit,
    remove,
  };
}

export type ConnectionFormController = ReturnType<typeof useConnectionForm>;
