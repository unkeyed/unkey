"use client";

import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import { zodResolver } from "@hookform/resolvers/zod";
import type { inferRouterInputs } from "@trpc/server";
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
  const utils = trpc.useUtils();
  const [confirmDelete, setConfirmDelete] = useState(false);

  const values = useMemo(() => drainToFormValues(drain), [drain]);

  const form = useForm<DrainFormValues>({
    resolver: zodResolver(editDrainSchema),
    defaultValues: emptyDrainForm,
    values,
    mode: "onChange",
  });

  const onUpdated = () => {
    utils.logdrain.list.invalidate();
    utils.logdrain.get.invalidate({ id: drain.id });
    toast.success("Log drain updated");
  };
  const update = trpc.logdrain.update.useMutation({
    onSuccess: onUpdated,
    onError: (error) => toast.error(error.message),
  });
  // Its own instance, so pausing from the menu does not put the panel's Save button into loading.
  const setStatus = trpc.logdrain.update.useMutation({
    onSuccess: onUpdated,
    onError: (error) => toast.error(error.message),
  });

  const remove = trpc.logdrain.delete.useMutation({
    onSuccess: () => {
      utils.logdrain.list.invalidate();
      toast.success("Log drain deleted");
      setConfirmDelete(false);
      onDeleted();
    },
    onError: (error) => toast.error(error.message),
  });

  const save = (onSaved: () => void) =>
    form.handleSubmit((submitted) => {
      const name = submitted.name.trim();
      const destination = changedDestination(submitted, values);
      const eventTypes = submittedEventTypes(submitted);
      const eventTypesChanged = !sameEventTypes(eventTypes, values.eventTypes);
      const outcomesChanged = !sameEventTypes(submitted.outcomes, values.outcomes);
      const keySpacesChanged = !sameEventTypes(submitted.keySpaceIds, values.keySpaceIds);
      const statusClasses = submittedStatusClasses(submitted);
      const sources = submittedSources(submitted);
      const statusesChanged = !sameEventTypes(statusClasses, values.statusClasses);
      const projectsChanged = !sameEventTypes(sources.projectIds, values.projectIds);
      const appsChanged = !sameEventTypes(sources.appIds, values.appIds);
      const environmentsChanged = !sameEventTypes(sources.environmentIds, values.environmentIds);
      if (
        name === drain.name &&
        destination === undefined &&
        !eventTypesChanged &&
        !outcomesChanged &&
        !keySpacesChanged &&
        !statusesChanged &&
        !projectsChanged &&
        !appsChanged &&
        !environmentsChanged
      ) {
        onSaved();
        return;
      }
      update.mutate(
        {
          id: drain.id,
          ...(name !== drain.name ? { name } : {}),
          ...(eventTypesChanged ? { eventTypes } : {}),
          ...(outcomesChanged ? { outcomes: submitted.outcomes } : {}),
          ...(keySpacesChanged ? { keySpaceIds: submitted.keySpaceIds } : {}),
          ...(statusesChanged ? { statusClasses } : {}),
          ...(projectsChanged ? { projectIds: sources.projectIds } : {}),
          ...(appsChanged ? { appIds: sources.appIds } : {}),
          ...(environmentsChanged ? { environmentIds: sources.environmentIds } : {}),
          ...(destination !== undefined ? { destination } : {}),
        },
        { onSuccess: onSaved },
      );
    });

  const toggleStatus = () =>
    setStatus.mutate({
      id: drain.id,
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

type UpdateDestination = inferRouterInputs<Router>["logdrain"]["update"]["destination"];

function sameEventTypes<T extends string | number>(left: T[], right: T[]): boolean {
  return left.length === right.length && left.every((eventType) => right.includes(eventType));
}

function changedDestination(
  submitted: DrainFormValues,
  current: DrainFormValues,
): UpdateDestination | undefined {
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
        kind: "http",
        config: {
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
        kind: "axiom",
        config: {
          ...(dataset !== current.dataset ? { dataset } : {}),
          ...(token !== "" ? { token: submitted.token } : {}),
        },
      };
    }
    default:
      throw new Error(`Unsupported log drain sink: ${submitted.kind satisfies never}`);
  }
}
