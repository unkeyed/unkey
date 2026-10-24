"use client";

import { Switch } from "@/components/ui/switch";
import {
  IconCircleInfoOutline18,
  IconCloudUploadOutline18,
  IconPlusOutline18,
  IconTrashOutline18,
} from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import {
  type ChangeEvent,
  type ClipboardEvent,
  type ComponentProps,
  type KeyboardEvent,
  type ReactNode,
  useId,
  useRef,
} from "react";
import { pastedEntries } from "../env-file";
import { DropOverlay } from "./drop-overlay";
import { type Field, useCellFocus } from "./use-cell-focus";
import { useDropZone } from "./use-drop-zone";
import type { VariableRows } from "./use-variable-rows";
import {
  type RowErrors,
  type RowPatch,
  type RowView,
  type VariableRow,
  rowView,
} from "./variable-rows";

const sensitiveHelp =
  "Hidden in the dashboard and the API after you save. Unkey encrypts every value.";

export function VariablesTable({ draft }: { draft: VariableRows }) {
  const { isDragging, importFile, dropZoneProps } = useDropZone(draft.importEntries);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const lastIndex = draft.rows.length - 1;
  const firstNew = draft.rows.findIndex((row) => row.origin.kind === "new");
  const errorIdPrefix = useId();
  const { cellRef, focusCell, footerRef } = useCellFocus(draft.rows.length - firstNew);

  const handleKeyPaste = (id: string, e: ClipboardEvent<HTMLInputElement>) => {
    const entries = pastedEntries(e.clipboardData.getData("text/plain"));
    if (entries.length === 0) {
      return;
    }
    e.preventDefault();
    draft.pasteAt(id, entries);
  };

  const handleKeyKeyDown = (index: number, e: KeyboardEvent<HTMLInputElement>) => {
    const previous = draft.rows[index - 1];
    if (e.key !== "Backspace" || e.currentTarget.value !== "" || previous?.origin.kind !== "new") {
      return;
    }
    e.preventDefault();
    draft.remove(draft.rows[index].id);
    focusCell(previous.id, "value");
  };

  const handleValueKeyDown = (index: number, e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey) && index === lastIndex) {
      e.preventDefault();
      focusCell(draft.add(), "key");
    }
  };

  const handleFileImport = (e: ChangeEvent<HTMLInputElement>) => {
    importFile(e.target.files?.[0]);
    e.target.value = "";
  };

  const footerActions = [
    { label: "Add variable", Icon: IconPlusOutline18, onClick: draft.add },
    {
      label: "Import .env",
      Icon: IconCloudUploadOutline18,
      onClick: () => fileInputRef.current?.click(),
    },
  ];

  return (
    <div
      {...dropZoneProps}
      tabIndex={-1}
      className="relative shrink-0 overflow-hidden rounded-lg border bg-raised outline-hidden [--variables-columns:minmax(0,2fr)_minmax(0,3fr)_96px_36px]"
    >
      <DropOverlay isDragging={isDragging} />
      <div className="grid grid-cols-(--variables-columns) divide-x border-b bg-grayA-2 text-xs text-gray-10">
        <span className="px-3 py-1.5">Key</span>
        <span className="px-3 py-1.5">Value</span>
        <span className="flex items-center gap-1 px-3 py-1.5">
          Sensitive
          <InfoTooltip content={sensitiveHelp} position={{ side: "top" }} className="max-w-64">
            <IconCircleInfoOutline18 className="size-3 text-gray-9" />
          </InfoTooltip>
        </span>
        <span />
      </div>
      <div className="divide-y">
        {draft.rows.map((row, index) => (
          <VariableRowView
            key={row.id}
            row={row}
            labels={rowLabels(row, index - firstNew + 1)}
            errors={draft.errors.get(row.id)}
            errorId={`${errorIdPrefix}-${row.id}`}
            cellRef={(field) => cellRef(row.id, field)}
            onChange={(patch) => draft.update(row.id, patch)}
            onRemove={() => draft.remove(row.id)}
            onKeyPaste={(e) => handleKeyPaste(row.id, e)}
            onKeyKeyDown={(e) => handleKeyKeyDown(index, e)}
            onValueKeyDown={(e) => handleValueKeyDown(index, e)}
          />
        ))}
      </div>
      <input
        ref={fileInputRef}
        type="file"
        accept=".env,.txt,text/plain"
        className="hidden"
        onChange={handleFileImport}
      />
      <div ref={footerRef} className="flex h-9 items-center justify-between border-t">
        {footerActions.map(({ label, Icon, onClick }) => (
          <Button
            key={label}
            type="button"
            variant="ghost"
            size="lg"
            onClick={onClick}
            className="rounded-none px-3 text-gray-11 hover:bg-grayA-2 hover:text-gray-12 focus:ring-0 focus-visible:bg-grayA-2 [&_svg]:size-3"
          >
            <Icon />
            {label}
          </Button>
        ))}
      </div>
    </div>
  );
}

type RowLabels = { key: string; value: string; sensitive: string; remove: string };

function rowLabels(row: VariableRow, position: number): RowLabels {
  if (row.origin.kind === "new") {
    return {
      key: `Key, row ${position}`,
      value: `Value, row ${position}`,
      sensitive: `Sensitive, row ${position}`,
      remove: `Remove variable, row ${position}`,
    };
  }
  return {
    key: row.key,
    value: `Value of ${row.key}`,
    sensitive: `${row.key} is sensitive`,
    remove: `Remove ${row.key}`,
  };
}

type RowProps = {
  row: VariableRow;
  labels: RowLabels;
  errors: RowErrors | undefined;
  errorId: string;
  cellRef: (field: Field) => (element: HTMLElement | null) => (() => void) | undefined;
  onChange: (patch: RowPatch) => void;
  onRemove: () => void;
  onKeyPaste: (e: ClipboardEvent<HTMLInputElement>) => void;
  onKeyKeyDown: (e: KeyboardEvent<HTMLInputElement>) => void;
  onValueKeyDown: (e: KeyboardEvent<HTMLTextAreaElement>) => void;
};

function ValueInput(props: ComponentProps<"textarea">) {
  return (
    <textarea
      rows={1}
      spellCheck={false}
      data-1p-ignore
      autoComplete="off"
      className="max-h-40 min-h-9 w-full min-w-0 resize-none bg-transparent px-3 py-2 font-mono text-sm leading-5 text-gray-12 outline-hidden [field-sizing:content] placeholder:text-grayA-8 focus:bg-grayA-2 aria-invalid:bg-error-2"
      {...props}
    />
  );
}

function SavedKey({ row }: RowProps) {
  return (
    <span className="flex min-h-9 min-w-0 items-center px-3 font-mono text-sm text-gray-12">
      <span className="truncate">{row.key}</span>
    </span>
  );
}

const KEY_CELL: Record<RowView, (props: RowProps) => ReactNode> = {
  new: ({ row, labels, errors, errorId, cellRef, onChange, onKeyPaste, onKeyKeyDown }) => (
    <div>
      <input
        aria-label={labels.key}
        aria-invalid={errors?.key ? true : undefined}
        aria-describedby={errors?.key ? errorId : undefined}
        placeholder="Key"
        spellCheck={false}
        data-1p-ignore
        autoComplete="off"
        className="h-9 w-full min-w-0 bg-transparent px-3 font-mono text-sm text-gray-12 outline-hidden placeholder:text-grayA-8 focus:bg-grayA-2 aria-invalid:bg-error-2"
        ref={cellRef("key")}
        value={row.key}
        onChange={(e) => onChange({ key: e.target.value })}
        onPaste={onKeyPaste}
        onKeyDown={onKeyKeyDown}
      />
    </div>
  ),
  none: SavedKey,
  sensitive: SavedKey,
  varies: SavedKey,
};

const VALUE_CELL: Record<RowView, (props: RowProps) => ReactNode> = {
  new: ({ row, labels, errors, errorId, cellRef, onChange, onValueKeyDown }) => (
    <ValueInput
      aria-label={labels.value}
      aria-invalid={errors?.value ? true : undefined}
      aria-describedby={errors?.value && !errors.key ? errorId : undefined}
      placeholder="Value"
      ref={cellRef("value")}
      value={row.value}
      onChange={(e) => onChange({ value: e.target.value })}
      onKeyDown={onValueKeyDown}
    />
  ),
  none: ({ row, labels, errors, errorId, onChange }) => (
    <ValueInput
      aria-label={labels.value}
      aria-invalid={errors?.value ? true : undefined}
      aria-describedby={errors?.value ? errorId : undefined}
      value={row.value}
      onChange={(e) => onChange({ value: e.target.value })}
    />
  ),
  sensitive: () => (
    <span className="flex min-h-9 items-center px-3 font-mono text-sm text-gray-10">
      ••••••••••••
    </span>
  ),
  varies: () => (
    <span className="flex min-h-9 items-center px-3 text-sm text-gray-10">
      Varies by environment
    </span>
  ),
};

function VariableRowView(props: RowProps) {
  const { row, labels, errors, errorId, onChange, onRemove } = props;
  const view = rowView(row);
  const KeyCell = KEY_CELL[view];
  const ValueCell = VALUE_CELL[view];
  return (
    <div>
      <div className="grid grid-cols-(--variables-columns) divide-x">
        <KeyCell {...props} />
        <ValueCell {...props} />
        <span className="flex px-3 pt-2">
          <Switch
            size="sm"
            aria-label={labels.sensitive}
            checked={row.sensitive}
            disabled={view !== "new"}
            onCheckedChange={(checked) => onChange({ sensitive: checked })}
          />
        </span>
        <span className="flex justify-center pt-1.5">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={labels.remove}
            onClick={onRemove}
          >
            <IconTrashOutline18 className="size-3.5 text-gray-9" />
          </Button>
        </span>
      </div>
      {errors ? (
        <p id={errorId} className="border-t bg-error-2 px-3 py-1.5 text-xs text-error-11">
          {errors.key ?? errors.value}
        </p>
      ) : null}
    </div>
  );
}
