"use client";

import { type LeaveConfirm, LeaveGuard, useLeaveBypass } from "@/components/leave-guard";
import { keepLeaveSentinel } from "@/hooks/prevent-leave-model";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useIsMutating, useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import {
  type Dispatch,
  type ReactNode,
  createContext,
  use,
  useEffect,
  useReducer,
  useRef,
  useState,
} from "react";
import { CardPending } from "./card";
import { FlowSplit, FlowViewport, PendingFlowCard } from "./flow-layout";
import { ConfigureSkeleton } from "./panes/configure-skeleton";
import { StepNav } from "./step-nav";
import { newAppErrorMessage, useAppLifecycle } from "./use-app-lifecycle";
import {
  type Resume,
  type SourceKind,
  type WizardAction,
  type WizardState,
  cardCopy,
  flowGuard,
  pendingPane,
  resumeWizard,
  wizardReducer,
  wizardSearchParams,
} from "./wizard-model";

type NewAppFlow = {
  projectId: string;
  state: WizardState;
  locked: boolean;
  dispatch: Dispatch<WizardAction>;
  leave: () => void;
  ensureApp: (source: SourceKind, create: () => Promise<string>) => Promise<string>;
};

export const NEW_APP_MUTATION_KEY = ["new-app-flow"];

const NewAppFlowContext = createContext<NewAppFlow | null>(null);

export function useNewAppFlow(): NewAppFlow {
  const flow = use(NewAppFlowContext);
  if (!flow) {
    throw new Error("useNewAppFlow must be used inside FlowLoader");
  }
  return flow;
}

export function FlowLoader({ projectId, children }: { projectId: string; children: ReactNode }) {
  const searchParams = useSearchParams();
  const params = { step: searchParams.get("step"), appId: searchParams.get("appId") };
  const { data: apps, isLoading } = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );
  const pending = pendingPane(params.step);
  if (params.appId && isLoading) {
    return (
      <PendingFlowCard card={pending}>
        {pending === "configure" ? (
          <ConfigureSkeleton source={null} />
        ) : (
          <CardPending message={cardCopy[pending].pending} />
        )}
      </PendingFlowCard>
    );
  }
  return (
    <ResumedFlow projectId={projectId} resume={() => resumeWizard(params, apps)}>
      {children}
    </ResumedFlow>
  );
}

function ResumedFlow({
  projectId,
  resume,
  children,
}: {
  projectId: string;
  resume: () => Resume;
  children: ReactNode;
}) {
  const [resumed] = useState(resume);
  if (resumed.kind === "app") {
    return <AppRedirect projectId={projectId} appId={resumed.appId} />;
  }
  return (
    <NewAppFlowProvider projectId={projectId} initial={resumed.state}>
      {children}
    </NewAppFlowProvider>
  );
}

function AppRedirect({ projectId, appId }: { projectId: string; appId: string }) {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const href = routes.projects.apps.overview({ workspaceSlug: workspace.slug, projectId, appId });
  useEffect(() => {
    router.replace(href);
  }, [router, href]);
  return null;
}

function wizardUrl(pathname: string, state: WizardState): string {
  const search = new URLSearchParams(wizardSearchParams(state)).toString();
  return search ? `${pathname}?${search}` : pathname;
}

function locationUrl(): string {
  return `${window.location.pathname}${window.location.search}`;
}

function replaceWizardUrl(url: string) {
  window.history.replaceState(keepLeaveSentinel(window.history.state), "", url);
}

function useHistorySync(state: WizardState) {
  const pathname = usePathname();
  useEffect(() => {
    const url = wizardUrl(pathname, state);
    if (locationUrl() !== url) {
      replaceWizardUrl(url);
    }
  }, [pathname, state]);

  useEffect(() => {
    const onPopState = () => {
      const url = wizardUrl(pathname, state);
      if (window.location.pathname === pathname && locationUrl() !== url) {
        replaceWizardUrl(url);
      }
    };
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
  }, [pathname, state]);
}

function useLeaving() {
  const [leaving, setLeaving] = useState(false);
  useEffect(() => {
    const onPageShow = (event: PageTransitionEvent) => {
      if (event.persisted) {
        setLeaving(false);
      }
    };
    window.addEventListener("pageshow", onPageShow);
    return () => window.removeEventListener("pageshow", onPageShow);
  }, []);
  return { leaving, markLeaving: () => setLeaving(true) };
}

const DISCARD_APP_COPY = {
  title: "Discard this app?",
  description: "If you leave now, you'll lose this app and its settings.",
  keepLabel: "Continue setup",
  discardLabel: "Discard app",
};

function useDiscardApp(projectId: string, appId: string | null): LeaveConfirm | undefined {
  const { deleteApp } = useAppLifecycle(projectId);
  const discard = useMutation({
    mutationFn: deleteApp,
    onError: (error) =>
      toast.error("Could not discard the app", { description: newAppErrorMessage(error) }),
  });
  if (appId === null) {
    return undefined;
  }
  return {
    copy: DISCARD_APP_COPY,
    loading: discard.isLoading,
    beforeLeave: () => discard.mutateAsync(appId),
  };
}

function NewAppFlowProvider({
  projectId,
  initial,
  children,
}: {
  projectId: string;
  initial: WizardState;
  children: ReactNode;
}) {
  const workspace = useWorkspaceNavigation();
  const [state, dispatch] = useReducer(wizardReducer, initial);
  const { leaving, markLeaving } = useLeaving();
  const mutating = useIsMutating({ mutationKey: NEW_APP_MUTATION_KEY }) > 0;
  const discardApp = useDiscardApp(projectId, state.appId);
  return (
    <LeaveGuard
      fallbackHref={routes.projects.detail({ workspaceSlug: workspace.slug, projectId })}
      mode={flowGuard({ hasApp: state.appId !== null, mutating, leaving })}
      confirm={discardApp}
    >
      <NewAppFlowContextProvider
        projectId={projectId}
        state={state}
        dispatch={dispatch}
        locked={mutating || leaving}
        markLeaving={markLeaving}
      >
        {children}
      </NewAppFlowContextProvider>
    </LeaveGuard>
  );
}

function NewAppFlowContextProvider({
  projectId,
  state,
  dispatch,
  locked,
  markLeaving,
  children,
}: {
  projectId: string;
  state: WizardState;
  dispatch: Dispatch<WizardAction>;
  locked: boolean;
  markLeaving: () => void;
  children: ReactNode;
}) {
  const bypass = useLeaveBypass();
  const leave = () => {
    bypass();
    markLeaving();
  };
  useHistorySync(state);
  const creating = useRef<Promise<string> | null>(null);

  const ensureApp: NewAppFlow["ensureApp"] = (source, create) => {
    if (state.appId) {
      return Promise.resolve(state.appId);
    }
    creating.current ??= create()
      .then((appId) => {
        dispatch({ type: "app-ready", appId, source });
        return appId;
      })
      .catch((error: unknown) => {
        creating.current = null;
        throw error;
      });
    return creating.current;
  };

  const flow: NewAppFlow = {
    projectId,
    state,
    locked,
    dispatch,
    leave,
    ensureApp,
  };
  return (
    <NewAppFlowContext.Provider value={flow}>
      <FlowViewport>
        <FlowSplit steps={<StepNav />}>{children}</FlowSplit>
      </FlowViewport>
    </NewAppFlowContext.Provider>
  );
}
