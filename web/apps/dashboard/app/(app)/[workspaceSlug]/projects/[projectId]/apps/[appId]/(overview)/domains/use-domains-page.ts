import { useResourceSearch } from "@/components/resource-search-input";
import { useConsumeSearchFlag } from "@/hooks/use-consume-search-flag";
import { collection } from "@/lib/collections";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { usePageLimit } from "@/lib/loadable/list";
import { match } from "@unkey/match";
import { useCallback, useState } from "react";
import { useProjectData } from "../data-provider";
import { domainsView } from "./model";
import type { AddDomainOutcome } from "./use-add-domain";

const PAGE_SIZE = 25;
const SEARCH_KEY = "search";

export function useDomainsPage() {
  const {
    environments,
    domains,
    customDomains,
    isDomainsLoading,
    isCustomDomainsLoading,
    isEnvironmentsLoading,
  } = useProjectData();
  const load = useCollectionLoad(collection.domains.utils, collection.customDomains.utils);
  const [search, setSearch] = useResourceSearch(SEARCH_KEY);
  const { limit, loadMore } = usePageLimit(search, PAGE_SIZE);
  const [addOpen, setAddOpen] = useState(false);
  const [limitMessage, setLimitMessage] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set());

  const defaultEnvironment =
    environments.find((e) => e.kind === "production") ?? environments.at(0);
  const openAdd = useCallback(() => {
    setLimitMessage(null);
    setAddOpen(true);
  }, []);

  const view = domainsView({
    isLoading: isDomainsLoading || isCustomDomainsLoading,
    failed: load.failed,
    domains,
    customDomains,
    environments,
    search,
    limit,
  });
  useConsumeSearchFlag(
    "add",
    view.state !== "loading" && !isEnvironmentsLoading,
    defaultEnvironment ? openAdd : null,
  );

  const toggle = (hostname: string) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      if (!next.delete(hostname)) {
        next.add(hostname);
      }
      return next;
    });

  return {
    view,
    environments,
    searchKey: SEARCH_KEY,
    expanded,
    toggle,
    loadMore,
    clearSearch: () => setSearch(null),
    retry: load.retry,
    limitMessage,
    add: {
      open: addOpen,
      defaultEnvironment,
      openAdd,
      onSettled: (outcome: AddDomainOutcome) => {
        setAddOpen(false);
        match(outcome)
          .with({ kind: "added" }, ({ domain }) => {
            setSearch(null);
            setExpanded((prev) => new Set(prev).add(domain));
          })
          .with({ kind: "limitReached" }, ({ message }) => setLimitMessage(message))
          .exhaustive();
      },
      onCancel: () => setAddOpen(false),
    },
  };
}
