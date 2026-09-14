import { ChevronLeft, ChevronRight, ListFilter } from "lucide-react";
import { Button } from "~/components/ui/button";
import { Input } from "~/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "~/components/ui/popover";
import { type FilterPopoverController, useFilterPopover } from "~/hooks/use-filter-popover";
import { cn } from "~/lib/utils";
import type { Dimension, FilterBarProps, FilterDimension } from "./filter-model";
import { appliedFilters } from "./filter-model";
import { DIMENSION_ICONS, FilterPill, Kbd, OptionRow, onListKeyDown } from "./filter-ui";

export function FilterBar({ dimensions }: FilterBarProps) {
  const filter = useFilterPopover(dimensions);
  const applied = appliedFilters(dimensions);

  return (
    <>
      <Popover open={filter.open} onOpenChange={filter.onOpenChange}>
        <PopoverTrigger
          render={
            <Button variant="outline" className="gap-2" aria-label="Add filter">
              <ListFilter className="text-gray-10" />
              Filter
            </Button>
          }
        />
        <PopoverContent
          align="start"
          className="w-[calc(100vw-2rem)] p-0 sm:w-64 md:w-auto"
          onKeyDown={onListKeyDown}
        >
          <div className="relative flex flex-col md:flex-row md:items-start">
            <div data-list="" className={cn("md:w-64", filter.panel && "hidden md:block")}>
              <div className="flex items-center gap-2 border-primary/10 border-b px-2 py-1.5">
                <Input
                  value={filter.query}
                  onChange={(event) => filter.setQuery(event.target.value)}
                  placeholder="Add filter…"
                  aria-label="Add filter"
                  className="h-6 border-0 px-1 shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
                />
                <Kbd>F</Kbd>
              </div>
              {filter.searching ? (
                <SearchHits filter={filter} />
              ) : (
                <DimensionList dimensions={dimensions} filter={filter} />
              )}
            </div>

            {filter.panel && <DimensionPanel panel={filter.panel} filter={filter} />}
          </div>
        </PopoverContent>
      </Popover>

      {applied.map((appliedFilter) => (
        <FilterPill key={appliedFilter.dimension} {...appliedFilter} />
      ))}
    </>
  );
}

function SearchHits({ filter }: { filter: FilterPopoverController }) {
  return (
    <div className="max-h-[min(20rem,50dvh)] overflow-y-auto overscroll-contain p-1">
      {filter.hits.map(({ dimension, options }) => (
        <div key={dimension.id}>
          {options.length > 0 && <GroupLabel>{dimension.groupLabel}</GroupLabel>}
          {options.map((option) => (
            <OptionRow
              key={option.value}
              label={option.label}
              count={option.count}
              swatch={option.swatch}
              checked={dimension.selected.includes(option.value)}
              onToggle={() => dimension.onToggle(option.value)}
            />
          ))}
        </div>
      ))}
      {filter.noHits && <p className="px-2 py-6 text-center text-gray-10 text-sm">No matches</p>}
    </div>
  );
}

function DimensionList({
  dimensions,
  filter,
}: {
  dimensions: FilterDimension[];
  filter: FilterPopoverController;
}) {
  return (
    <div className="p-1">
      {dimensions.map((dimension) => {
        const Icon = DIMENSION_ICONS[dimension.id];
        return (
          <button
            key={dimension.id}
            data-row=""
            type="button"
            onClick={() => filter.openPanel(dimension.id)}
            onMouseEnter={() => filter.hoverPanel(dimension.id)}
            onKeyDown={(event) => {
              if (event.key === "ArrowRight") {
                event.preventDefault();
                filter.openPanel(dimension.id);
              }
            }}
            className={cn(
              "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-gray-12 text-sm transition-colors hover:bg-gray-3 focus-visible:bg-gray-3 focus-visible:outline-none",
              filter.panel?.id === dimension.id && "bg-gray-3",
            )}
          >
            <Icon className="size-4 shrink-0 text-gray-10" aria-hidden="true" />
            <span className="flex-1 text-left">{dimension.label}</span>
            <ChevronRight className="size-3.5 shrink-0 text-gray-9" aria-hidden="true" />
          </button>
        );
      })}
    </div>
  );
}

function DimensionPanel({
  panel,
  filter,
}: {
  panel: FilterDimension;
  filter: FilterPopoverController;
}) {
  return (
    <div
      ref={filter.panelRef}
      data-list=""
      className="border-primary/10 border-t md:absolute md:top-0 md:left-full md:ml-1 md:w-72 md:rounded-md md:border md:border-gray-6 md:bg-background md:shadow-md"
    >
      <button
        type="button"
        onClick={filter.closePanel}
        className="flex w-full items-center gap-1.5 border-primary/10 border-b px-2 py-2 text-gray-11 text-sm hover:text-gray-12 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-12 md:hidden"
      >
        <ChevronLeft className="size-4" aria-hidden="true" />
        {panel.label}
      </button>
      <div className="border-primary/10 border-b px-2 py-1.5">
        <Input
          value={filter.panelQuery}
          onChange={(event) => filter.setPanelQuery(event.target.value)}
          placeholder="Filter…"
          aria-label={`Filter ${panel.groupLabel.toLowerCase()}`}
          className="h-6 border-0 px-1 shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
        />
      </div>
      <div className="max-h-[min(20rem,50dvh)] overflow-y-auto overscroll-contain p-1">
        {filter.visibleOptions.map((option) => (
          <OptionRow
            key={option.value}
            label={option.label}
            count={option.count}
            swatch={option.swatch}
            checked={panel.selected.includes(option.value)}
            onToggle={() => panel.onToggle(option.value)}
          />
        ))}
        {filter.noPanelHits && (
          <p className="px-2 py-6 text-center text-gray-10 text-sm">No matches</p>
        )}
      </div>
      {filter.emptyOptions.length > 0 && (
        <button
          type="button"
          onClick={filter.toggleShowEmpty}
          className="flex w-full items-center gap-2 border-primary/10 border-t px-3 py-2 text-left text-gray-10 text-xs hover:text-gray-11 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-12"
        >
          <PanelIcon dimension={panel.id} />
          {filter.showEmpty
            ? "Hide options with no requests"
            : `${filter.emptyOptions.length} options not matching any requests`}
        </button>
      )}
    </div>
  );
}

function PanelIcon({ dimension }: { dimension: Dimension }) {
  const Icon = DIMENSION_ICONS[dimension];
  return <Icon className="size-3.5 shrink-0" aria-hidden="true" />;
}

function GroupLabel({ children }: { children: React.ReactNode }) {
  return <p className="px-2 pt-2 pb-1 font-medium text-gray-10 text-xs">{children}</p>;
}
