import { getRouteApi } from "@tanstack/react-router";
import { type KeysPageContent, deriveKeysPage } from "~/components/keys-page/keys-page-model";
import { useKeysSearch } from "~/hooks/use-keys-search";
import { useKeysUsage } from "~/hooks/use-keys-usage";
import { type RotateKeyController, useRotateKey } from "~/hooks/use-rotate-key";
import { useTimeWindow } from "~/hooks/use-time-window";
import { isUnauthorizedError } from "~/lib/portal-api";
import { canRerollKeys } from "~/lib/scopes";

export type KeysPageView =
  | { status: "session-expired"; returnUrl: string | null }
  | ({
      status: "ready";
      title: string;
      rotate: RotateKeyController;
      canRotate: boolean;
    } & KeysPageContent);

const route = getRouteApi("/_portal/keys");

export function useKeysPage(): KeysPageView {
  const { session, portal, logsRetentionDays } = route.useRouteContext();

  const search = useKeysSearch(logsRetentionDays);
  const timeWindow = useTimeWindow(search.preset);
  const usage = useKeysUsage(timeWindow, session.scopes);
  const rotate = useRotateKey();

  const expired =
    isUnauthorizedError(usage.keysState.error) || isUnauthorizedError(usage.aggregateState.error);
  if (expired) {
    return { status: "session-expired", returnUrl: session.returnUrl };
  }

  const content = deriveKeysPage({
    usage,
    selectedKeys: search.selectedKeys,
    selectedOutcomes: search.selectedOutcomes,
    selectedStatus: search.selectedStatus,
    presets: search.presets,
    preset: search.preset,
    defaultPreset: search.defaultPreset,
    patch: search.patch,
    days: timeWindow.days,
  });

  return {
    status: "ready",
    title: portal?.displayName ? `${portal.displayName} API` : "API keys",
    rotate,
    canRotate: canRerollKeys(session.scopes),
    ...content,
  };
}
