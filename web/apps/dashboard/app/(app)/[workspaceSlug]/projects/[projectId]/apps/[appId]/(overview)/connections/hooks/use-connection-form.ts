"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { type Callers, connectedAppIds } from "../model/callers";
import {
  type ConnectionFormValues,
  connectionFormSchema,
  openForm,
  takenNames,
  targetsForApp,
} from "../model/form";
import { FORM_MODES } from "../model/mode";
import { connectionHostVariable, defaultConnectionName } from "../model/name";
import { planSave, removalOf, removesPreview, renameOf } from "../model/plan";
import { needsRedeploy } from "../model/run";
import { firstStepWithError, openStep, pickedApp } from "../model/steps";
import type { ListedConnection } from "../model/target";
import { connectedTargets } from "../model/target-options";
import { whoGetsLines } from "../model/who-gets";
import type { ConnectionMutation } from "./use-connection-actions";
import type { CallerSnapshot } from "./use-connections-data";

export function useConnectionForm({
  callers,
  before,
  mutation,
  onClose,
}: {
  callers: Callers<CallerSnapshot>;
  before: ListedConnection | null;
  mutation: ConnectionMutation;
  onClose: () => void;
}) {
  const { apps } = callers.production.targets;
  const [opening] = useState(() => openForm(before, callers));
  const { intent } = opening;
  const mode = intent.kind;
  const form = useForm<ConnectionFormValues>({
    resolver: zodResolver(connectionFormSchema(callers, intent)),
    defaultValues: opening.values,
  });
  const [steps, setSteps] = useState(FORM_MODES[mode].initialSteps);
  const [targetAppId, name, targets] = useWatch({
    control: form.control,
    name: ["targetAppId", "name", "targets"],
  });
  const target = before
    ? { id: before.targetAppId, name: before.targetAppName }
    : apps.find((app) => app.id === targetAppId);
  const planned = planSave(callers, { targetAppId, name, targets }, intent);

  const pickApp = (id: string) => {
    const appChanged = id !== form.getValues("targetAppId");
    setSteps((state) => pickedApp(state, appChanged));
    if (!appChanged) {
      return;
    }
    form.setValue("targets", targetsForApp(callers, form.getValues(), id), { shouldDirty: true });
    form.setValue("targetAppId", id, { shouldDirty: true, shouldValidate: true });
    const picked = apps.find((app) => app.id === id);
    if (picked && !form.getFieldState("name").isDirty) {
      const taken = takenNames(callers, id, planSave(callers, form.getValues(), intent));
      form.setValue("name", defaultConnectionName(picked.slug, (slug) => taken.has(slug)) ?? "");
    }
  };

  const finish = (submitted?: ConnectionFormValues) => {
    form.reset(submitted);
    onClose();
  };

  const commit = () =>
    form.handleSubmit(
      (submitted) => {
        const ops = planSave(callers, submitted, intent);
        if (ops.length === 0) {
          finish(submitted);
          return;
        }
        const renamed = intent.kind === "edit" ? renameOf(intent.opened, submitted.name) : null;
        mutation.run(
          {
            action: mode,
            ops,
            description: renamed
              ? `Update your code to use ${connectionHostVariable(renamed)}.`
              : undefined,
          },
          ({ failure }) => {
            if (!failure) {
              finish(submitted);
            }
          },
        );
      },
      (errors) => {
        const step = firstStepWithError(errors);
        if (step) {
          setSteps((state) => openStep(state, step));
        }
      },
    )();

  const remove = () => {
    if (!before) {
      return;
    }
    mutation.run(
      { action: "remove", ops: removalOf(callers, before.targetAppId) },
      ({ applied, failure }) => {
        if (!failure || applied.length > 0) {
          finish();
        }
      },
    );
  };

  const { redeployNote } = FORM_MODES[mode];

  return {
    mode,
    before,
    form,
    steps,
    setSteps,
    target,
    callers,
    name: name.trim(),
    targets,
    connected: target ? connectedTargets(callers, targets, target.name) : [],
    whoGets: target ? whoGetsLines(planned, callers, target.name) : [],
    alsoPreview: before ? removesPreview(removalOf(callers, before.targetAppId)) : false,
    redeployNote: redeployNote && needsRedeploy(planned) ? redeployNote : null,
    apps,
    connectedIds: connectedAppIds(callers),
    pending: mutation.pending,
    pickApp,
    commit,
    remove,
  };
}

export type ConnectionFormController = ReturnType<typeof useConnectionForm>;
