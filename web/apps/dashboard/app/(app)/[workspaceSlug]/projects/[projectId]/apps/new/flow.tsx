"use client";

import { TOP_NAV_HEIGHT } from "@/components/navigation/top-nav";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Skeleton } from "@unkey/ui";
import { usePathname, useSearchParams } from "next/navigation";
import {
  type CSSProperties,
  type Dispatch,
  type ReactNode,
  createContext,
  useContext,
  useEffect,
  useReducer,
} from "react";
import { GithubConnectingCard } from "./github-connecting";
import type { CreateAppResult } from "./use-app-lifecycle";
import {
  type Card,
  type CardId,
  GITHUB_RETURN_STEP,
  type SourceKind,
  type WizardAction,
  type WizardState,
  cardList,
  flowLocked,
  resolveCard,
  resumeFromApps,
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
    if (returningFromGithub) {
      return <GithubConnectingCard />;
    }
    return (
      <div className="flex items-center justify-center p-10" style={viewportHeight}>
        <Skeleton className="h-full w-full max-w-[960px] rounded-xl" />
      </div>
    );
  }
  return (
    <NewAppFlowProvider
      projectId={projectId}
      initial={() => resumeFromApps(params, apps)}
      returningFromGithub={returningFromGithub}
    >
      {children}
    </NewAppFlowProvider>
  );
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
  initial: () => WizardState;
  returningFromGithub: boolean;
  children: ReactNode;
}) {
  const [state, dispatch] = useReducer(wizardReducer, undefined, initial);
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
