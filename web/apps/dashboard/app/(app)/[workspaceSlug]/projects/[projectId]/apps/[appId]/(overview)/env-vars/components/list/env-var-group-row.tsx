import { IconChevronRightOutline12 } from "@unkey/icons";
import { Badge, InfoTooltip } from "@unkey/ui";
import { cn } from "cn";
import { HighlightMatch } from "../shared/highlight-match";
import { EnvVarBaseRow } from "./env-var-base-row";
import { EnvVarGroupActionMenu } from "./env-var-group-action-menu";
import { type DisplayRow, EnvVarItemRow } from "./env-var-item-row";

type GroupRowProps = {
  row: DisplayRow & { kind: "group" };
  isExpanded: boolean;
  selected: boolean | "partial";
  selectedIds: Set<string>;
  deferredQuery: string;
  editingId: string | null;
  onToggleGroup: () => void;
  onToggleSelection: (shiftKey: boolean) => void;
  onToggleItemSelection: (itemId: string) => void;
  onEdit: (id: string) => void;
  onCloseEdit: () => void;
  hasSelection: boolean;
};

export function GroupRow({
  row,
  isExpanded,
  selected,
  selectedIds,
  deferredQuery,
  editingId,
  onToggleGroup,
  onToggleSelection,
  onToggleItemSelection,
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
              <InfoTooltip content={row.key} position={{ side: "top" }} asChild>
                <span className="font-mono font-medium text-sm text-gray-12 truncate leading-4 max-w-[250px]">
                  <HighlightMatch text={row.key} query={deferredQuery} />
                </span>
              </InfoTooltip>
              {row.hasWriteonly && (
                <Badge
                  className="px-1.5 py-0 rounded-md h-5 text-2xs font-medium pointer-events-none"
                  variant="warning"
                >
                  Sensitive
                </Badge>
              )}
            </div>
            <div className="text-sm mt-1 text-gray-11 capitalize">All Environments</div>
          </div>
        </div>
      }
      valueCell={
        <span className="flex items-center gap-1.5 text-sm text-gray-11 transition-colors pl-2">
          {row.items.length} values
          <IconChevronRightOutline12
            className={cn(
              "size-[12px] transition-transform duration-200",
              isExpanded && "rotate-90",
            )}
          />
        </span>
      }
      timestamp={row.latestCreatedAt}
      actionsCell={<EnvVarGroupActionMenu groupKey={row.key} items={row.items} />}
      expandedContent={
        isExpanded ? (
          <div className="divide-y divide-grayA-3 bg-grayA-2 border-t">
            {row.items.map((item) => (
              <EnvVarItemRow
                key={item.id}
                item={item}
                searchQuery={deferredQuery}
                isEditing={editingId === item.id}
                onEdit={() => onEdit(item.id)}
                onCloseEdit={onCloseEdit}
                isSelected={selectedIds.has(item.id)}
                onToggleSelection={() => onToggleItemSelection(item.id)}
                hasSelection={hasSelection}
              />
            ))}
          </div>
        ) : undefined
      }
    />
  );
}
