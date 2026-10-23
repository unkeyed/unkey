import { IconChevronRightOutline12, IconMagnifierOutline18 } from "@unkey/icons";
import { Button, Checkbox, InputGroup, InputGroupAddon, InputGroupInput } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useId, useMemo, useState } from "react";

type FilterLike = {
  field: string;
  value: string | number;
};

export type CheckboxGroup = {
  id: string;
  label: string;
  items: { value: string; label: string }[];
};

type Props<TFilter extends FilterLike> = {
  groups: CheckboxGroup[];
  filterField: TFilter["field"];
  filters: TFilter[];
  updateFilters: (filters: TFilter[]) => void;
  createFilter: (value: string) => TFilter;
  searchPlaceholder?: string;
};

export function GroupedFilterCheckbox<TFilter extends FilterLike>({
  groups,
  filterField,
  filters,
  updateFilters,
  createFilter,
  searchPlaceholder = "Search...",
}: Props<TFilter>) {
  const id = useId();
  const [search, setSearch] = useState("");
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(() => new Set());
  const [draft, setDraft] = useState<ReadonlySet<string>>(
    () =>
      new Set(
        filters
          .filter((filter) => filter.field === filterField)
          .map((filter) => String(filter.value)),
      ),
  );

  const term = search.trim().toLowerCase();
  const visibleGroups = useMemo(
    () =>
      groups
        .map((group) => ({
          ...group,
          items: group.label.toLowerCase().includes(term)
            ? group.items
            : group.items.filter((item) =>
                `${group.label}.${item.label}`.toLowerCase().includes(term),
              ),
        }))
        .filter((group) => group.items.length > 0),
    [groups, term],
  );

  const setSelected = (values: string[], selected: boolean) => {
    setDraft((previous) => {
      const next = new Set(previous);
      for (const value of values) {
        if (selected) {
          next.add(value);
        } else {
          next.delete(value);
        }
      }
      return next;
    });
  };

  const toggleExpanded = (groupId: string) => {
    setExpanded((previous) => {
      const next = new Set(previous);
      if (!next.delete(groupId)) {
        next.add(groupId);
      }
      return next;
    });
  };

  const applyFilter = () => {
    updateFilters([
      ...filters.filter((filter) => filter.field !== filterField),
      ...[...draft].map(createFilter),
    ]);
  };

  return (
    <div className="flex w-80 flex-col overflow-hidden">
      <div className="border-b p-2">
        <InputGroup variant="ghost" className="h-8">
          <InputGroupAddon className="pointer-events-none">
            <IconMagnifierOutline18 className="size-3.5 text-gray-9" />
          </InputGroupAddon>
          <InputGroupInput
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder={searchPlaceholder}
            aria-label={searchPlaceholder}
            className="h-8 text-xs"
          />
        </InputGroup>
      </div>

      <div className="max-h-80 min-h-40 overflow-y-auto p-2 [scrollbar-width:thin]">
        {visibleGroups.length === 0 ? (
          <div className="px-2 py-8 text-center text-xs text-gray-9">No matches</div>
        ) : (
          <div className="flex flex-col gap-0.5">
            {visibleGroups.map((group) => {
              const values = group.items.map((item) => item.value);
              const selectedCount = values.filter((value) => draft.has(value)).length;
              const checkboxState =
                selectedCount === 0
                  ? false
                  : selectedCount === values.length
                    ? true
                    : "indeterminate";
              const isOpen = term !== "" || expanded.has(group.id);

              return (
                <div key={group.id}>
                  <div
                    className={cn(
                      "flex h-9 items-center gap-1 rounded-md pr-2 hover:bg-grayA-3",
                      checkboxState !== false && "bg-grayA-3",
                    )}
                  >
                    <button
                      type="button"
                      onClick={() => toggleExpanded(group.id)}
                      className="flex size-6 shrink-0 items-center justify-center rounded text-gray-9 hover:text-gray-12"
                      aria-label={`${isOpen ? "Collapse" : "Expand"} ${group.label}`}
                      aria-expanded={isOpen}
                    >
                      <IconChevronRightOutline12
                        className={cn("size-3 transition-transform", isOpen && "rotate-90")}
                      />
                    </button>
                    <Checkbox
                      checked={checkboxState}
                      onCheckedChange={() => setSelected(values, checkboxState !== true)}
                      className="size-4 shrink-0 rounded-sm [&_svg]:size-3"
                      aria-label={`Select all ${group.label}`}
                    />
                    <button
                      type="button"
                      onClick={() => toggleExpanded(group.id)}
                      className="flex min-w-0 flex-1 items-center justify-between pl-2 text-left"
                    >
                      <span className="truncate font-mono text-xs font-medium text-gray-12">
                        {group.label}
                      </span>
                      <span className="text-2xs tabular-nums text-gray-9">
                        {selectedCount > 0 ? `${selectedCount}/${values.length}` : values.length}
                      </span>
                    </button>
                  </div>
                  {isOpen ? (
                    <div className="ml-[34px] border-l py-1 pl-2">
                      {group.items.map((item) => (
                        <label
                          key={item.value}
                          htmlFor={`${id}-${item.value}`}
                          className="flex h-8 cursor-pointer items-center gap-2 rounded-md px-2 hover:bg-grayA-3"
                        >
                          <Checkbox
                            id={`${id}-${item.value}`}
                            checked={draft.has(item.value)}
                            onCheckedChange={() =>
                              setSelected([item.value], !draft.has(item.value))
                            }
                            className="size-4 shrink-0 rounded-sm [&_svg]:size-3"
                            aria-label={`${group.label} ${item.label}`}
                          />
                          <span className="truncate font-mono text-xs text-gray-12">
                            {item.label}
                          </span>
                        </label>
                      ))}
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
        )}
      </div>

      <div className="border-t p-2">
        <Button variant="primary" className="h-9 w-full rounded-md" onClick={applyFilter}>
          Apply Filter
        </Button>
      </div>
    </div>
  );
}
