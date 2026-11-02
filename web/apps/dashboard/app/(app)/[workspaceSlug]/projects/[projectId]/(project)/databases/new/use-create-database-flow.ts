"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { readyOr } from "@/lib/loadable/loadable";
import { routes } from "@/lib/navigation/routes";
import { toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useEffect, useReducer, useState } from "react";
import { useCreateDatabase, usePlanetscale, useProjectDatabases } from "../use-project-databases";
import { type Step, deriveFlow, flowReducer, initialFlow } from "./create-flow-model";

const CONNECT_DELAY_MS = 1_000;

export function useCreateDatabaseFlow() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const project = useProjectDatabases();
  const planetscale = usePlanetscale();
  const [state, dispatch] = useReducer(flowReducer, initialFlow);
  const create = useCreateDatabase();
  const [connecting, setConnecting] = useState(false);
  const scope = { workspaceSlug: workspace.slug, projectId: project.projectId };
  const submitting = create.isLoading || create.isSuccess;
  const view = deriveFlow(state, {
    ...planetscale,
    linked: readyOr(project.databases, []),
  });

  useEffect(() => {
    if (!connecting) {
      return;
    }
    const timer = window.setTimeout(() => {
      planetscale.connect();
      dispatch({ type: "step", step: "source" });
      setConnecting(false);
      toast.success("Connected to PlanetScale");
    }, CONNECT_DELAY_MS);
    return () => window.clearTimeout(timer);
  }, [connecting, planetscale.connect]);

  const goTo = (step: Step | null) => {
    if (step) {
      dispatch({ type: "step", step });
    }
  };

  const submit = () => {
    if (submitting || !view.connected || !view.input) {
      return;
    }
    create.mutate(view.input, {
      onSuccess: (db) => {
        router.replace(routes.projects.databases.list({ ...scope, database: db.id }));
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : "Could not add the database.");
      },
    });
  };

  return {
    state,
    view,
    dispatch,
    connecting,
    submitting,
    cancelHref: routes.projects.databases.list(scope),
    goTo,
    advance: () => goTo(view.next),
    connect: () => setConnecting(true),
    disconnect: planetscale.disconnect,
    submit,
  };
}

export type DatabaseFlow = ReturnType<typeof useCreateDatabaseFlow>;
