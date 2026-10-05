"use client";

import { TOP_NAV_HEIGHT } from "@/components/navigation/top-nav";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { Skeleton } from "@unkey/ui";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import {
  type CSSProperties,
  type Dispatch,
  type ReactNode,
  createContext,
  useContext,
  useEffect,
  useReducer,
  useState,
} from "react";
import { GithubConnectingCard } from "./github-connecting";
import type { CreateAppResult } from "./use-app-lifecycle";
import {
  type Card,
  type CardId,
  GITHUB_RETURN_STEP,
  type Resume,
  type SourceKind,
  type WizardAction,
  type WizardState,
  cardList,
  flowLocked,
  resolveCard,
  resumeWizard,
  wizardReducer,
  wizardSearchParams,
} from "./wizard-model";

export const viewportHeight: CSSProperties = { height: `calc(100dvh - ${TOP_NAV_HEIGHT}px)` };

type NewAppFlow = {
  projectId: string;
  state: WizardState;
  card: Card;
  cards: readonly CardId[];
  locked: boolean;
  returningFromGithub: boolean;
  dispatch: Dispatch<WizardAction>;
  ensureApp: (
    source: SourceKind,
    create: () => Promise<CreateAppResult>,
  ) => Promise<CreateAppResult>;
};

const NewAppFlowContext = createContext<NewAppFlow | null>(null);

export function useNewAppFlow(): NewAppFlow {
  const flow = useContext(NewAppFlowContext);
  if (!flow) {
    throw new Error("useNewAppFlow must be used inside FlowLoader");
  }
  return flow;
}

export function useApp(projectId: string, appId: string) {
  const { data } = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  return data.at(0);
}

export function FlowLoader({ projectId, children }: { projectId: string; children: ReactNode }) {
  const searchParams = useSearchParams();
  const params = {
    step: searchParams.get("step"),
    appId: searchParams.get("appId"),
    deploymentId: searchParams.get("deploymentId"),
  };
  const returningFromGithub = params.step === GITHUB_RETURN_STEP;
  const { data: apps, isLoading } = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );
  if (params.appId && isLoading) {
    return returningFromGithub ? <GithubConnectingCard /> : flowSkeleton;
  }
  return (
    <ResumedFlow
      projectId={projectId}
      resume={() => resumeWizard(params, apps)}
      returningFromGithub={returningFromGithub}
    >
      {children}
    </ResumedFlow>
  );
}

const flowSkeleton = (
  <div className="flex items-center justify-center p-10" style={viewportHeight}>
    <Skeleton className="h-full w-full max-w-[960px] rounded-xl" />
  </div>
);

// The decision is taken once: the URL and the app list both change as the flow
// deploys, and a later mismatch must not send the user away mid-flow.
function ResumedFlow({
  projectId,
  resume,
  returningFromGithub,
  children,
}: {
  projectId: string;
  resume: () => Resume;
  returningFromGithub: boolean;
  children: ReactNode;
}) {
  const [resumed] = useState(resume);
  if (resumed.kind === "app") {
    return <AppRedirect projectId={projectId} appId={resumed.appId} />;
  }
  return (
    <NewAppFlowProvider
      projectId={projectId}
      initial={resumed.state}
      returningFromGithub={returningFromGithub}
    >
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
  return flowSkeleton;
}

function useSyncSearchParams(state: WizardState) {
  const pathname = usePathname();
  const search = new URLSearchParams(wizardSearchParams(state)).toString();
  useEffect(() => {
    const url = search ? `${pathname}?${search}` : pathname;
    if (`${window.location.pathname}${window.location.search}` !== url) {
      window.history.replaceState(null, "", url);
    }
  }, [pathname, search]);
}

function NewAppFlowProvider({
  projectId,
  initial,
  returningFromGithub,
  children,
}: {
  projectId: string;
  initial: WizardState;
  returningFromGithub: boolean;
  children: ReactNode;
}) {
  const [state, dispatch] = useReducer(wizardReducer, initial);
  useSyncSearchParams(state);

  const ensureApp: NewAppFlow["ensureApp"] = async (source, create) => {
    if (state.app) {
      return { ok: true, appId: state.app.id };
    }
    dispatch({ type: "create-start" });
    const created = await create();
    dispatch(
      created.ok ? { type: "app-ready", appId: created.appId, source } : { type: "create-failed" },
    );
    return created;
  };

  const flow: NewAppFlow = {
    projectId,
    state,
    card: resolveCard(state),
    cards: cardList(state),
    locked: flowLocked(state),
    returningFromGithub,
    dispatch,
    ensureApp,
  };
  return <NewAppFlowContext.Provider value={flow}>{children}</NewAppFlowContext.Provider>;
}
