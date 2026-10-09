import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import type { Environment } from "@/lib/collections/deploy/environments";
import { IconCodeOutline12, IconLockOutline12 } from "@unkey/icons";
import { Checkbox, InfoTooltip, ResourceListRow, TimestampInfo } from "@unkey/ui";
import { cn } from "cn";
import { EnvironmentLabel } from "../../environment-label";
import { EnvVarActionMenu } from "./env-var-action-menu";
import { EnvVarEditRow } from "./env-var-edit-row";
import { EnvVarNameCell } from "./env-var-name-cell";
import { EnvVarValueCell } from "./env-var-value-cell";

export type EnvVarItem = EnvVar & { environment: Environment | undefined };

export function compareByName(a: EnvVarItem, b: EnvVarItem): number {
  return (
    a.key.localeCompare(b.key) ||
    (a.environment?.slug ?? "").localeCompare(b.environment?.slug ?? "")
  );
}

type RowSelection = {
  isSelected: boolean;
  hasSelection: boolean;
  onToggle: (shiftKey: boolean) => void;
};

type EnvVarRowProps = {
  item: EnvVarItem;
  searchQuery: string;
  isEditing: boolean;
  onEdit: () => void;
  onCloseEdit: () => void;
  selection?: RowSelection;
};

export function EnvVarRow({
  item,
  searchQuery,
  isEditing,
  onEdit,
  onCloseEdit,
  selection,
}: EnvVarRowProps) {
  return (
    <div>
      <ResourceListRow
        label={`Edit ${item.key}`}
        expanded={isEditing}
        onActivate={isEditing ? onCloseEdit : onEdit}
        className="group/row grid grid-cols-[28px_minmax(0,1.3fr)_minmax(0,1.2fr)_minmax(0,0.7fr)_minmax(0,0.7fr)_20px] items-center gap-4 text-xs"
      >
        {selection ? (
          <SelectableTypeMarker envVar={item} selection={selection} />
        ) : (
          <TypeMarker envVar={item} />
        )}
        <span className="flex min-w-0 justify-self-start">
          <EnvVarNameCell envVar={item} searchQuery={searchQuery} />
        </span>
        <span className="flex min-w-0 justify-self-start">
          <EnvVarValueCell envVar={item} />
        </span>
        <span className="flex min-w-0 items-center">
          <EnvironmentLabel environment={item.environment} />
        </span>
        <span className="relative flex min-w-0 items-center gap-1 justify-self-start text-xs text-gray-11">
          Added
          <TimestampInfo
            displayType="relative"
            value={item.createdAt}
            side="top"
            className="truncate text-gray-11 hover:text-gray-12"
          />
        </span>
        <span className="relative flex justify-end">
          <EnvVarActionMenu envVar={item} onEdit={onEdit} />
        </span>
      </ResourceListRow>
      {isEditing ? (
        <div className="grid animate-expand-down overflow-hidden">
          <div className="min-h-0">
            <EnvVarEditRow envVar={item} onClose={onCloseEdit} />
          </div>
        </div>
      ) : null}
    </div>
  );
}

const TYPE_MARKERS = {
  writeonly: { Icon: IconLockOutline12, label: "Sensitive. You can't view the value." },
  recoverable: { Icon: IconCodeOutline12, label: "Not sensitive. You can view the value." },
} satisfies Record<EnvVar["type"], { Icon: typeof IconLockOutline12; label: string }>;

function TypeMarker({ envVar, className }: { envVar: EnvVar; className?: string }) {
  const { Icon, label } = TYPE_MARKERS[envVar.type];

  return (
    <InfoTooltip content={label} position={{ side: "top" }} asChild>
      <span
        className={cn(
          "relative flex size-6 items-center justify-center rounded-md border bg-raised text-gray-12 transition-opacity",
          className,
        )}
      >
        <Icon />
        <span className="sr-only">{label}</span>
      </span>
    </InfoTooltip>
  );
}

function SelectableTypeMarker({ envVar, selection }: { envVar: EnvVar; selection: RowSelection }) {
  const { isSelected, hasSelection, onToggle } = selection;

  return (
    // biome-ignore lint/a11y/useKeyWithClickEvents: the checkbox handles keyboard interaction
    <span
      className="group/marker relative flex size-6 shrink-0 items-center justify-center"
      onClick={(e) => {
        if (e.target instanceof HTMLInputElement) {
          return;
        }
        onToggle(e.shiftKey);
      }}
    >
      <TypeMarker
        envVar={envVar}
        className={hasSelection ? "opacity-0" : "group-hover/marker:opacity-0"}
      />
      <Checkbox
        checked={isSelected}
        aria-label={`Select ${envVar.key}`}
        className={cn(
          "absolute size-4 [&_svg]:size-3",
          hasSelection
            ? "opacity-100"
            : "opacity-0 group-hover/marker:opacity-100 focus-visible:opacity-100",
        )}
      />
    </span>
  );
}
