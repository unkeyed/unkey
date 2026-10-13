"use client";

import { Switch } from "@/components/ui/switch";
import {
  IconCircleInfoOutline18,
  IconCloudUploadOutline18,
  IconPlusOutline18,
  IconTrashOutline18,
} from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { type ChangeEvent, type ClipboardEvent, type KeyboardEvent, useId, useRef } from "react";
import { pastedEntries } from "../env-file";
import { DropOverlay } from "./drop-overlay";
import { useCellFocus } from "./use-cell-focus";
import { useDropZone } from "./use-drop-zone";
import type { VariableRows } from "./use-variable-rows";

const sensitiveHelp =
  "Hidden in the dashboard and the API after you save. Unkey encrypts every value.";

export function VariablesTable({ draft }: { draft: VariableRows }) {
  const { isDragging, importFile, dropZoneProps } = useDropZone(draft.importEntries);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const lastIndex = draft.rows.length - 1;
  const errorIdPrefix = useId();
  const { cellRef, focusCell, footerRef } = useCellFocus(draft.rows.length);

  const handleKeyPaste = (id: string, e: ClipboardEvent<HTMLInputElement>) => {
    const entries = pastedEntries(e.clipboardData.getData("text/plain"));
    if (entries.length === 0) {
      return;
    }
    e.preventDefault();
    draft.pasteAt(id, entries);
  };

  const handleKeyKeyDown = (index: number, e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Backspace" || draft.rows.length === 1 || e.currentTarget.value !== "") {
      return;
    }
    e.preventDefault();
    const previous = draft.rows[index - 1];
    if (!previous) {
      return;
    }
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
      className="relative overflow-hidden rounded-lg border border-grayA-4 bg-raised outline-hidden [--variables-columns:minmax(0,2fr)_minmax(0,3fr)_96px_36px]"
    >
      <DropOverlay isDragging={isDragging} />
      <div className="grid grid-cols-(--variables-columns) divide-x divide-grayA-4 border-b border-grayA-4 bg-grayA-2 text-xs text-gray-10">
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
      <div className="divide-y divide-grayA-4">
        {draft.rows.map((row, index) => {
          const rowErrors = draft.errors.get(row.id);
          const errorId = `${errorIdPrefix}-${row.id}`;
          return (
            <div key={row.id}>
              <div className="grid grid-cols-(--variables-columns) divide-x divide-grayA-4">
                <div>
                  <input
                    aria-label={`Key, row ${index + 1}`}
                    aria-invalid={rowErrors?.key ? true : undefined}
                    aria-describedby={rowErrors?.key ? errorId : undefined}
                    placeholder="Key"
                    spellCheck={false}
                    data-1p-ignore
                    autoComplete="off"
                    className="h-9 w-full min-w-0 bg-transparent px-3 font-mono text-sm text-gray-12 outline-hidden placeholder:text-grayA-8 focus:bg-grayA-2 aria-invalid:bg-error-2"
                    ref={cellRef(row.id, "key")}
                    value={row.key}
                    onChange={(e) => draft.update(row.id, { key: e.target.value })}
                    onPaste={(e) => handleKeyPaste(row.id, e)}
                    onKeyDown={(e) => handleKeyKeyDown(index, e)}
                  />
                </div>
                <textarea
                  rows={1}
                  aria-label={`Value, row ${index + 1}`}
                  aria-invalid={rowErrors?.value ? true : undefined}
                  aria-describedby={rowErrors?.value && !rowErrors.key ? errorId : undefined}
                  placeholder="Value"
                  spellCheck={false}
                  data-1p-ignore
                  autoComplete="off"
                  className="max-h-40 min-h-9 w-full min-w-0 resize-none bg-transparent px-3 py-2 font-mono text-sm leading-5 text-gray-12 outline-hidden [field-sizing:content] placeholder:text-grayA-8 focus:bg-grayA-2 aria-invalid:bg-error-2"
                  ref={cellRef(row.id, "value")}
                  value={row.value}
                  onChange={(e) => draft.update(row.id, { value: e.target.value })}
                  onKeyDown={(e) => handleValueKeyDown(index, e)}
                />
                <span className="flex px-3 pt-2">
                  <Switch
                    size="sm"
                    aria-label={`Sensitive, row ${index + 1}`}
                    checked={row.sensitive}
                    onCheckedChange={(checked) => draft.update(row.id, { sensitive: checked })}
                  />
                </span>
                <span className="flex justify-center pt-1.5">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    aria-label={`Remove variable, row ${index + 1}`}
                    onClick={() => draft.remove(row.id)}
                  >
                    <IconTrashOutline18 className="size-3.5 text-gray-9" />
                  </Button>
                </span>
              </div>
              {rowErrors ? (
                <p
                  id={errorId}
                  className="border-t border-grayA-4 bg-error-2 px-3 py-1.5 text-xs text-error-11"
                >
                  {rowErrors.key ?? rowErrors.value}
                </p>
              ) : null}
            </div>
          );
        })}
      </div>
      <input
        ref={fileInputRef}
        type="file"
        accept=".env,.txt,text/plain"
        className="hidden"
        onChange={handleFileImport}
      />
      <div
        ref={footerRef}
        className="flex h-9 items-center justify-between border-t border-grayA-4"
      >
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
