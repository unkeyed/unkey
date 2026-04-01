import { EnvVarActionMenu } from "./env-var-action-menu";
import { EnvVarBaseRow } from "./env-var-base-row";
import { EnvVarEditRow } from "./env-var-edit-row";
import { EnvVarNameCell } from "./env-var-name-cell";
import { EnvVarValueCell } from "./env-var-value-cell";

export { TimestampBadge } from "./env-var-base-row";

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
  const showCheckbox = selectable && !!onToggleSelection;

  return (
    <EnvVarBaseRow
      showCheckbox={showCheckbox}
      checked={isSelected}
      forceCheckboxVisible={isSelected || hasSelection}
      onCheckboxClick={showCheckbox ? (shiftKey) => onToggleSelection(shiftKey) : undefined}
      nameCell={
        <EnvVarNameCell
          envVarId={item.id}
          variableKey={item.key}
          environmentName={item.environmentName}
          note={item.note}
          searchQuery={searchQuery}
          type={item.type}
        />
      }
      valueCell={<EnvVarValueCell envVarId={item.id} type={item.type} />}
      timestamp={item.updatedAt}
      actionsCell={
        <EnvVarActionMenu
          envVarId={item.id}
          variableKey={item.key}
          type={item.type}
          onEdit={onEdit}
        />
      }
      expandedContent={
        isEditing ? (
          <EnvVarEditRow
            environmentId={item.environmentId}
            envVarId={item.id}
            variableKey={item.key}
            type={item.type}
            note={item.note ?? null}
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
