import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import { EnvVarActionMenu } from "./env-var-action-menu";
import { EnvVarBaseRow } from "./env-var-base-row";
import { EnvVarEditRow } from "./env-var-edit-row";
import { EnvVarNameCell } from "./env-var-name-cell";
import { EnvVarValueCell } from "./env-var-value-cell";

export { TimestampBadge } from "./env-var-base-row";

export type EnvVarItem = EnvVar & { environmentName: string };

export type DisplayRow =
  | { kind: "single"; item: EnvVarItem }
  | {
      kind: "group";
      key: string;
      items: EnvVarItem[];
      latestCreatedAt: number;
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
  const showCheckbox = selectable && !!onToggleSelection;

  return (
    <EnvVarBaseRow
      showCheckbox={showCheckbox}
      checked={isSelected}
      forceCheckboxVisible={isSelected || hasSelection}
      onCheckboxClick={showCheckbox ? (shiftKey) => onToggleSelection(shiftKey) : undefined}
      onRowClick={isEditing ? onCloseEdit : onEdit}
      nameCell={
        <EnvVarNameCell
          value={item.value}
          variableKey={item.key}
          environmentName={item.environmentName}
          note={item.description}
          searchQuery={searchQuery}
          type={item.type}
        />
      }
      valueCell={<EnvVarValueCell value={item.value} type={item.type} />}
      timestamp={item.createdAt}
      actionsCell={
        <EnvVarActionMenu
          envVarId={item.id}
          value={item.value}
          variableKey={item.key}
          type={item.type}
          onEdit={onEdit}
        />
      }
      expandedContent={
        isEditing ? (
          <EnvVarEditRow
            envVarId={item.id}
            value={item.value}
            variableKey={item.key}
            type={item.type}
            note={item.description}
            onClose={onCloseEdit}
          />
        ) : undefined
      }
    />
  );
}

export function rowKey(r: DisplayRow): string {
  return r.kind === "group" ? r.key : r.item.key;
}

export function rowTime(r: DisplayRow): number {
  return r.kind === "group" ? r.latestCreatedAt : r.item.createdAt;
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
        latestCreatedAt: Math.max(...group.map((i) => i.createdAt)),
        hasWriteonly: group.some((i) => i.type === "writeonly"),
      });
    }
  }
  return rows;
}
