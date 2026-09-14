"use client";

import { useDeleteLogdrainMutation, useUpdateLogdrainMutation } from "@/lib/logdrains-query";
import { getErrorMessage } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import type { LogdrainDestinationWrite } from "@unkey/api/models/components";
import { toast } from "@unkey/ui";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import {
  type DrainDetail,
  type DrainFormValues,
  drainToFormValues,
  editDrainSchema,
  emptyDrainForm,
  submittedEventTypes,
  submittedSources,
  submittedStatusClasses,
} from "../drain-schema";

export function useDrainSettings(drain: DrainDetail, { onDeleted }: { onDeleted: () => void }) {
  const [confirmDelete, setConfirmDelete] = useState(false);

  const values = useMemo(() => drainToFormValues(drain), [drain]);

  const form = useForm<DrainFormValues>({
    resolver: zodResolver(editDrainSchema),
    defaultValues: emptyDrainForm,
    values,
    mode: "onChange",
  });

  const onUpdated = () => {
    toast.success("Log drain updated");
  };
  const update = useUpdateLogdrainMutation({
    onSuccess: onUpdated,
    onError: (error) => toast.error(getErrorMessage(error)),
  });
  // Its own instance, so pausing from the menu does not put the panel's Save button into loading.
  const setStatus = useUpdateLogdrainMutation({
    onSuccess: onUpdated,
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const remove = useDeleteLogdrainMutation({
    onSuccess: () => {
      toast.success("Log drain deleted");
      setConfirmDelete(false);
      onDeleted();
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const save = (onSaved: () => void) =>
    form.handleSubmit((submitted) => {
      const name = submitted.name.trim();
      const destination = changedDestination(submitted, values);
      const eventTypes = submittedEventTypes(submitted);
      const eventTypesChanged = !sameEventTypes(eventTypes, values.eventTypes);
      const outcomesChanged = !sameEventTypes(submitted.outcomes, values.outcomes);
      const keySpacesChanged = !sameEventTypes(submitted.keySpaceIds, values.keySpaceIds);
      const severitiesChanged = !sameEventTypes(submitted.severities, values.severities);
      const namespacesChanged = !sameEventTypes(submitted.namespaceIds, values.namespaceIds);
      const passedChanged = !sameEventTypes(submitted.passed, values.passed);
      const projectField = drain.stream === "runtime_logs" ? "runtimeProjectIds" : "projectIds";
      const appField = drain.stream === "runtime_logs" ? "runtimeAppIds" : "appIds";
      const environmentField =
        drain.stream === "runtime_logs" ? "runtimeEnvironmentIds" : "environmentIds";
      const statusClasses = submittedStatusClasses(submitted);
      const sources = submittedSources(submitted);
      const statusesChanged = !sameEventTypes(statusClasses, values.statusClasses);
      const projectsChanged = !sameEventTypes(sources.projectIds, values[projectField]);
      const appsChanged = !sameEventTypes(sources.appIds, values[appField]);
      const environmentsChanged = !sameEventTypes(sources.environmentIds, values[environmentField]);
      if (
        name === drain.name &&
        destination === undefined &&
        !eventTypesChanged &&
        !outcomesChanged &&
        !keySpacesChanged &&
        !statusesChanged &&
        !severitiesChanged &&
        !namespacesChanged &&
        !passedChanged &&
        !projectsChanged &&
        !appsChanged &&
        !environmentsChanged
      ) {
        onSaved();
        return;
      }
      const filters = {
        ...(eventTypesChanged ? { eventTypes } : {}),
        ...(outcomesChanged ? { outcomes: submitted.outcomes } : {}),
        ...(keySpacesChanged ? { keySpaceIds: submitted.keySpaceIds } : {}),
        ...(severitiesChanged ? { severities: submitted.severities } : {}),
        ...(namespacesChanged ? { namespaceIds: submitted.namespaceIds } : {}),
        ...(passedChanged ? { passed: submitted.passed } : {}),
        ...(statusesChanged ? { statusClasses } : {}),
        ...(projectsChanged ? { projectIds: sources.projectIds } : {}),
        ...(appsChanged ? { appIds: sources.appIds } : {}),
        ...(environmentsChanged ? { environmentIds: sources.environmentIds } : {}),
      };
      update.mutate(
        {
          logdrainId: drain.id,
          ...(name !== drain.name ? { name } : {}),
          ...(Object.keys(filters).length > 0 ? { filters } : {}),
          ...(destination !== undefined ? { destination } : {}),
        },
        { onSuccess: onSaved },
      );
    });

  const toggleStatus = () =>
    setStatus.mutate({
      logdrainId: drain.id,
      status: drain.status === "running" ? "paused_by_user" : "running",
    });

  return {
    form,
    reset: () => form.reset(values),
    save,
    update,
    setStatus,
    remove,
    toggleStatus,
    confirmDelete,
    setConfirmDelete,
  };
}

export type DrainSettings = ReturnType<typeof useDrainSettings>;

function sameEventTypes<T extends string | number | boolean>(left: T[], right: T[]): boolean {
  return left.length === right.length && left.every((eventType) => right.includes(eventType));
}

function changedDestination(
  submitted: DrainFormValues,
  current: DrainFormValues,
): LogdrainDestinationWrite | undefined {
  switch (submitted.kind) {
    case "http": {
      const headers = submitted.headers.filter((header) => header.name.trim() !== "");
      const headersChanged =
        headers.length !== current.headers.length ||
        headers.some(
          (header, index) =>
            header.name.trim() !== current.headers[index]?.name || header.value !== "",
        );
      const url = submitted.url.trim();
      if (url === current.url && submitted.format === current.format && !headersChanged) {
        return undefined;
      }
      return {
        http: {
          ...(url !== current.url ? { url } : {}),
          ...(submitted.format !== current.format ? { format: submitted.format } : {}),
          ...(headersChanged
            ? {
                headers: headers.map((header) =>
                  header.stored && header.value === ""
                    ? { mode: "preserve" as const, name: header.name.trim() }
                    : {
                        mode: "set" as const,
                        name: header.name.trim(),
                        value: header.value,
                      },
                ),
              }
            : {}),
        },
      };
    }
    case "axiom": {
      const dataset = submitted.dataset.trim();
      const token = submitted.token.trim();
      if (dataset === current.dataset && token === "") {
        return undefined;
      }
      return {
        axiom: {
          ...(dataset !== current.dataset ? { dataset } : {}),
          ...(token !== "" ? { token: submitted.token } : {}),
        },
      };
    }
    default:
      throw new Error(`Unsupported log drain sink: ${submitted.kind satisfies never}`);
  }
}
