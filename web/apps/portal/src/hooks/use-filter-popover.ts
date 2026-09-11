import { useRef, useState } from "react";
import {
  type Dimension,
  type FilterDimension,
  searchHits,
  splitPanelOptions,
} from "~/components/keys-table/filters/filter-model";
import { useFilterShortcut } from "~/hooks/use-filter-shortcut";

export function useFilterPopover(dimensions: FilterDimension[]) {
  const [open, setOpen] = useState(false);
  const [panelId, setPanelId] = useState<Dimension | null>(null);
  const [query, setQuery] = useState("");
  const [panelQuery, setPanelQuery] = useState("");
  const [showEmpty, setShowEmpty] = useState(false);
  const panelRef = useRef<HTMLDivElement | null>(null);
  useFilterShortcut(setOpen);

  const panel = dimensions.find((dimension) => dimension.id === panelId);
  const hits = searchHits(dimensions, query);
  const options = splitPanelOptions(panel, panelQuery, showEmpty);

  const resetSearch = () => {
    setQuery("");
    setPanelQuery("");
    setShowEmpty(false);
  };

  const openPanel = (id: Dimension) => {
    setPanelId(id);
    resetSearch();
    requestAnimationFrame(() =>
      panelRef.current?.querySelector<HTMLElement>("[data-row]")?.focus(),
    );
  };

  // Hovering another dimension swaps the panel, but not out from under someone
  // who is already typing or arrowing through the open one.
  const hoverPanel = (id: Dimension) => {
    if (id === panelId || panelRef.current?.contains(document.activeElement)) {
      return;
    }
    setPanelId(id);
    resetSearch();
  };

  return {
    open,
    query,
    panelQuery,
    searching: query.trim() !== "",
    hits,
    noHits: hits.every((hit) => hit.options.length === 0),
    panel,
    visibleOptions: options.visible,
    emptyOptions: options.empty,
    noPanelHits: options.all.length === 0,
    showEmpty,
    panelRef,
    openPanel,
    hoverPanel,
    onOpenChange: (next: boolean) => {
      setOpen(next);
      if (!next) {
        setPanelId(null);
        resetSearch();
      }
    },
    closePanel: () => setPanelId(null),
    toggleShowEmpty: () => setShowEmpty(!showEmpty),
    setQuery: (value: string) => {
      setQuery(value);
      setPanelId(null);
    },
    setPanelQuery: (value: string) => setPanelQuery(value),
  };
}

export type FilterPopoverController = ReturnType<typeof useFilterPopover>;
