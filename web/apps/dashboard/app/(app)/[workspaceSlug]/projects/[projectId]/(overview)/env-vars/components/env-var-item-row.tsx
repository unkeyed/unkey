import { cn } from "@/lib/utils";
import { ChartActivity2 } from "@unkey/icons";
import { Badge, Checkbox, TimestampInfo } from "@unkey/ui";
import { EnvVarActionMenu } from "./env-var-action-menu";
import { EnvVarEditRow } from "./env-var-edit-row";
import { EnvVarNameCell } from "./env-var-name-cell";
import { EnvVarValueCell } from "./env-var-value-cell";

export type EnvVarItem = {
  id: string;
  key: string;
  environmentId: string;
  environmentName: string;
  type: "writeonly" | "recoverable";
  updatedAt: number;
  note: string | null | undefined;
};

export type DisplayRow =
  | { kind: "single"; item: EnvVarItem }
  | {
    kind: "group";
    key: string;
    items: EnvVarItem[];
    latestUpdatedAt: number;
    hasWriteonly: boolean;
  };

type EnvVarItemRowProps = {
  item: EnvVarItem;
  searchQuery: string;
  isEditing: boolean;
  onEdit: () => void;
  onCloseEdit: () => void;
  isSelected?: boolean;
  onToggleSelection?: (shiftKey: boolean) => void;
  selectable?: boolean;
  hasSelection?: boolean;
};

export function EnvVarItemRow({
  item,
  searchQuery,
  isEditing,
  onEdit,
  onCloseEdit,
  isSelected = false,
  onToggleSelection,
  selectable = true,
  hasSelection = false,
}: EnvVarItemRowProps) {
  const showCheckbox = selectable && onToggleSelection;

  return (
    <div>
      <div
        className="group flex items-center hover:bg-grayA-2 transition-colors"
      >
        {/* biome-ignore lint/a11y/useKeyWithClickEvents: checkbox handles keyboard interaction */}
        <div
          className="pl-4 flex items-center w-8 shrink-0"
          onClick={(e) => {
            if (showCheckbox) {
              e.stopPropagation();
              onToggleSelection(e.shiftKey);
            }
          }}
        >
          {showCheckbox && (
            <Checkbox
              checked={isSelected}
              className={cn(
                "size-4 [&_svg]:size-3",
                isSelected || hasSelection
                  ? "opacity-100"
                  : "opacity-0 pointer-events-none group-hover:opacity-100 group-hover:pointer-events-auto focus-visible:opacity-100 focus-visible:pointer-events-auto",
              )}
              onCheckedChange={() => { }}
            />
          )}
        </div>
        <div className="flex-4 min-w-0 py-3.5 flex items-center">
          <EnvVarNameCell
            envVarId={item.id}
            variableKey={item.key}
            environmentName={item.environmentName}
            note={item.note}
            searchQuery={searchQuery}
            type={item.type}
          />
        </div>
        <div className="flex-4 min-w-0 py-3.5 flex items-center pr-3">
          <EnvVarValueCell envVarId={item.id} type={item.type} />
        </div>
        <div className="flex-2 min-w-0 py-3.5 flex items-center pr-3">
          <TimestampBadge value={item.updatedAt} />
        </div>
        <div className="w-12 shrink-0 py-3.5 pr-4 flex items-center justify-end">
          <EnvVarActionMenu
            envVarId={item.id}
            variableKey={item.key}
            type={item.type}
            onEdit={onEdit}
          />
        </div>
      </div>
      {isEditing && (
        <div className="grid animate-expand-down overflow-hidden">
          <div className="min-h-0">
            <EnvVarEditRow
              environmentId={item.environmentId}
              envVarId={item.id}
              variableKey={item.key}
              type={item.type}
              note={item.note ?? null}
              onClose={onCloseEdit}
            />
          </div>
        </div>
      )}
    </div>
  );
}

export function TimestampBadge({ value }: { value: number }) {
  return (
    <Badge className="px-1.5 rounded-md flex gap-2 items-center h-5.5 border-none bg-grayA-3 text-grayA-12 truncate">
      <ChartActivity2 iconSize="sm-regular" className="shrink-0" />
      <TimestampInfo displayType="relative" value={value} className="truncate" />
    </Badge>
  );
}

export function rowKey(r: DisplayRow): string {
  return r.kind === "group" ? r.key : r.item.key;
}

export function rowTime(r: DisplayRow): number {
  return r.kind === "group" ? r.latestUpdatedAt : r.item.updatedAt;
}

export function groupByKey(items: EnvVarItem[]): DisplayRow[] {
  const groups = new Map<string, EnvVarItem[]>();
  for (const item of items) {
    const existing = groups.get(item.key);
    if (existing) {
      existing.push(item);
    } else {
      groups.set(item.key, [item]);
    }
  }

  const rows: DisplayRow[] = [];
  for (const [key, group] of groups) {
    if (group.length === 1) {
      rows.push({ kind: "single", item: group[0] });
    } else {
      group.sort((a, b) => a.environmentName.localeCompare(b.environmentName));
      rows.push({
        kind: "group",
        key,
        items: group,
        latestUpdatedAt: Math.max(...group.map((i) => i.updatedAt)),
        hasWriteonly: group.some((i) => i.type === "writeonly"),
      });
    }
  }
  return rows;
}
