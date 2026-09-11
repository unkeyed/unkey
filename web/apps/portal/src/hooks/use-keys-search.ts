import { getRouteApi } from "@tanstack/react-router";
import { useMemo } from "react";
import { z } from "zod";
import { OUTCOME_FILTER_KINDS } from "~/components/analytics/outcomes";
import {
  TIME_PRESET_IDS,
  availableTimePresets,
  defaultTimePreset,
} from "~/components/analytics/time-presets";
import { KEY_STATUSES } from "~/components/keys-table/key-status";

export const keysSearchSchema = z.object({
  keys: z.array(z.string()).optional().catch(undefined),
  outcomes: z.array(z.enum(OUTCOME_FILTER_KINDS)).optional().catch(undefined),
  status: z.array(z.enum(KEY_STATUSES)).optional().catch(undefined),
  since: z.enum(TIME_PRESET_IDS).optional().catch(undefined),
});

export type KeysSearch = z.infer<typeof keysSearchSchema>;

export type KeysSearchPatch = (next: Partial<KeysSearch>) => void;

const route = getRouteApi("/_portal/keys");

function orUndefined<T>(list: T[] | undefined): T[] | undefined {
  return list === undefined || list.length === 0 ? undefined : list;
}

/**
 * The keys page's URL state. An empty selection is written as an absent param
 * rather than an empty array, so clearing a filter leaves no trace in the URL.
 */
export function useKeysSearch(logsRetentionDays: number) {
  const search = route.useSearch();
  const navigate = route.useNavigate();

  const presets = useMemo(() => availableTimePresets(logsRetentionDays), [logsRetentionDays]);
  const defaultPreset = defaultTimePreset(logsRetentionDays);

  const patch: KeysSearchPatch = (next) =>
    navigate({
      search: (prev) => {
        const merged = { ...prev, ...next };
        return {
          ...merged,
          keys: orUndefined(merged.keys),
          outcomes: orUndefined(merged.outcomes),
          status: orUndefined(merged.status),
        };
      },
      replace: true,
    });

  return {
    selectedKeys: search.keys ?? [],
    selectedOutcomes: search.outcomes ?? [],
    selectedStatus: search.status ?? [],
    presets,
    defaultPreset,
    preset: presets.find((p) => p.id === search.since) ?? defaultPreset,
    patch,
  };
}
