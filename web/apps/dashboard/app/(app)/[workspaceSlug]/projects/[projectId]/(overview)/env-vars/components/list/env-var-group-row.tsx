import { Badge } from "@unkey/ui";
import { HighlightMatch } from "../shared/highlight-match";
import { EnvVarBaseRow } from "./env-var-base-row";
import { type DisplayRow, EnvVarItemRow } from "./env-var-item-row";

type GroupRowProps = {
  row: DisplayRow & { kind: "group" };
  isExpanded: boolean;
  selected: boolean | "partial";
  deferredQuery: string;
  editingId: string | null;
  onToggleGroup: () => void;
  onToggleSelection: (shiftKey: boolean) => void;
  onEdit: (id: string) => void;
  onCloseEdit: () => void;
  hasSelection: boolean;
};

export function GroupRow({
  row,
  isExpanded,
  selected,
  deferredQuery,
  editingId,
  onToggleGroup,
  onToggleSelection,
  onEdit,
  onCloseEdit,
  hasSelection,
}: GroupRowProps) {
  const isChecked = selected === true;
  const isIndeterminate = selected === "partial";

  return (
    <EnvVarBaseRow
      showCheckbox
      checked={isIndeterminate ? "indeterminate" : isChecked}
      forceCheckboxVisible={isChecked || isIndeterminate || hasSelection}
      onCheckboxClick={(shiftKey) => onToggleSelection(shiftKey)}
      onRowClick={onToggleGroup}
      nameCell={
        <div className="flex items-center px-4">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-1.5">
              <span className="font-mono font-medium text-[13px] text-accent-12 truncate leading-4 max-w-[250px]">
                <HighlightMatch text={row.key} query={deferredQuery} />
              </span>
              {row.hasWriteonly && (
                <Badge
                  className="px-1.5 py-0 rounded-md h-5 text-[11px] font-medium pointer-events-none"
                  variant="warning"
                >
                  Sensitive
                </Badge>
              )}
            </div>
            <div className="text-[13px] mt-1 text-gray-11 capitalize">All Environments</div>
          </div>
        </div>
      }
      valueCell={
        <span className="text-[13px] text-gray-11 transition-colors pl-2">
          {row.items.length} values ›
        </span>
      }
      timestamp={row.latestUpdatedAt}
      expandedContent={
        isExpanded ? (
          <div className="divide-y divide-grayA-3 bg-grayA-2 border-t border-grayA-4">
            {row.items.map((item) => (
              <EnvVarItemRow
                key={item.id}
                item={item}
                searchQuery={deferredQuery}
                isEditing={editingId === item.id}
                onEdit={() => onEdit(item.id)}
                onCloseEdit={onCloseEdit}
                selectable={false}
              />
            ))}
          </div>
        ) : undefined
      }
    />
  );
}
