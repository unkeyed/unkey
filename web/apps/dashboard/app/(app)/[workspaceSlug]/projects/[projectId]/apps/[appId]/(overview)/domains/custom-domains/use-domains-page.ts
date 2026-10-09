import { useResourceSearch } from "@/components/resource-search-input";
import { collection } from "@/lib/collections";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { match } from "@unkey/match";
import { useState } from "react";
import { useProjectData } from "../../data-provider";
import { domainsView } from "../model";
import type { AddDomainOutcome } from "./use-add-domain";

const PAGE_SIZE = 25;
const SEARCH_KEY = "search";

export function useDomainsPage() {
  const { environments, domains, customDomains, isDomainsLoading, isCustomDomainsLoading } =
    useProjectData();
  const load = useCollectionLoad(collection.domains.utils, collection.customDomains.utils);
  const [search, setSearch] = useResourceSearch(SEARCH_KEY);
  const [page, setPage] = useState({ search, limit: PAGE_SIZE });
  const limit = page.search === search ? page.limit : PAGE_SIZE;
  const [addOpen, setAddOpen] = useState(false);
  const [limitMessage, setLimitMessage] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set());

  const view = domainsView({
    isLoading: isDomainsLoading || isCustomDomainsLoading,
    failed: load.failed,
    domains,
    customDomains,
    environments,
    search,
    limit,
  });

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
    loadMore: () => setPage({ search, limit: limit + PAGE_SIZE }),
    clearSearch: () => setSearch(null),
    retry: load.retry,
    limitMessage,
    add: {
      open: addOpen,
      defaultEnvironment: environments.find((e) => e.kind === "production") ?? environments.at(0),
      openAdd: () => {
        setLimitMessage(null);
        setAddOpen(true);
      },
      onSettled: (outcome: AddDomainOutcome) => {
        setAddOpen(false);
        match(outcome)
          .with({ kind: "added" }, ({ domain }) => {
            setSearch(null);
            setExpanded((prev) => new Set(prev).add(domain));
          })
          .with({ kind: "limitReached" }, ({ message }) => setLimitMessage(message))
          .with({ kind: "cancelled" }, () => {})
          .exhaustive();
      },
    },
  };
}
