"use client";

import { Switch } from "@/components/ui/switch";
import {
  IconCircleInfoOutline18,
  IconCloudUploadOutline18,
  IconPlusOutline18,
  IconTrashOutline18,
} from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { cn } from "cn";
import {
  type ChangeEvent,
  type ClipboardEvent,
  type KeyboardEvent,
  useEffect,
  useId,
  useRef,
} from "react";
import { pastedEntries } from "../../env-file";
import { useDropZone } from "./use-drop-zone";
import type { VariableRows } from "./use-variable-rows";

const sensitiveHelp =
  "Hidden in the dashboard and the API after you save. Unkey encrypts every value.";

const columns = "grid grid-cols-[minmax(0,2fr)_minmax(0,3fr)_96px_36px] divide-x divide-grayA-4";

const cellInput =
  "h-9 w-full min-w-0 bg-transparent px-3 font-mono text-sm text-gray-12 outline-hidden placeholder:text-grayA-8 focus:bg-grayA-2 aria-invalid:bg-error-2";

const footerButton =
  "flex h-full items-center gap-2 px-3 text-sm font-medium text-gray-11 hover:bg-grayA-2 hover:text-gray-12 focus-visible:bg-grayA-2 focus-visible:outline-hidden";

type Field = "key" | "value";

const REVEAL_MARGIN_PX = 48;

function scrollParent(element: HTMLElement): HTMLElement | null {
  for (let node = element.parentElement; node; node = node.parentElement) {
    if (/(auto|scroll)/.test(getComputedStyle(node).overflowY)) {
      return node;
    }
  }
  return null;
}

function revealFooter(element: HTMLElement) {
  const scroller = scrollParent(element);
  if (!scroller) {
    return;
  }
  const hidden =
    element.getBoundingClientRect().bottom +
    REVEAL_MARGIN_PX -
    scroller.getBoundingClientRect().bottom;
  if (hidden > 0) {
    scroller.scrollBy({ top: hidden });
  }
}

export function VariablesTable({ draft }: { draft: VariableRows }) {
  const { ref, isDragging, importFile } = useDropZone<HTMLDivElement>(draft.importEntries);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const lastIndex = draft.rows.length - 1;
  const footerRef = useRef<HTMLDivElement>(null);
  const rowCountRef = useRef(draft.rows.length);
  const errorIdPrefix = useId();
  const cells = useRef(new Map<string, HTMLElement>());

  useEffect(() => {
    if (draft.rows.length > rowCountRef.current && footerRef.current) {
      revealFooter(footerRef.current);
    }
    rowCountRef.current = draft.rows.length;
  }, [draft.rows.length]);

  const cellRef = (id: string, field: Field) => (element: HTMLElement | null) => {
    if (!element) {
      return;
    }
    const cellKey = `${id}:${field}`;
    cells.current.set(cellKey, element);
    return () => {
      cells.current.delete(cellKey);
    };
  };

  const focusCell = (id: string, field: Field) => {
    requestAnimationFrame(() => {
      cells.current.get(`${id}:${field}`)?.focus();
    });
  };

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
    const file = e.target.files?.[0];
    if (file) {
      importFile(file);
    }
    e.target.value = "";
  };

  return (
    <div
      ref={ref}
      tabIndex={-1}
      className="relative overflow-hidden rounded-lg border border-grayA-4 bg-raised outline-hidden"
    >
      <div
        className={cn(
          "pointer-events-none absolute inset-0 z-10 flex items-center justify-center transition-all duration-200",
          isDragging ? "bg-successA-2 opacity-100" : "opacity-0",
        )}
      >
        <div
          className={cn(
            "absolute inset-2 rounded-md border-2 border-dashed transition-all duration-200",
            isDragging ? "scale-100 border-successA-8" : "scale-[0.98] border-transparent",
          )}
        />
        <div
          className={cn(
            "flex items-center gap-3 transition-all duration-200",
            isDragging ? "scale-100 opacity-100" : "scale-95 opacity-0",
          )}
        >
          <div className="flex size-8 items-center justify-center rounded-lg bg-successA-3">
            <IconCloudUploadOutline18 className="text-success-11" />
          </div>
          <span className="text-sm font-medium text-success-11">Drop your .env file</span>
        </div>
      </div>
      <div className={cn(columns, "border-b border-grayA-4 bg-grayA-2 text-xs text-gray-10")}>
        <span className="px-3 py-1.5">Key</span>
        <span className="px-3 py-1.5">Value</span>
        <span className="flex items-center gap-1 px-3 py-1.5">
          Sensitive
          <InfoTooltip content={sensitiveHelp} position={{ side: "top" }} className="z-60 max-w-64">
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
              <div className={columns}>
                <div>
                  <input
                    aria-label={`Key, row ${index + 1}`}
                    aria-invalid={rowErrors?.key ? true : undefined}
                    aria-describedby={rowErrors?.key ? errorId : undefined}
                    placeholder="Key"
                    spellCheck={false}
                    data-1p-ignore
                    autoComplete="off"
                    className={cellInput}
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
                  className={cn(
                    cellInput,
                    "h-auto max-h-40 min-h-9 resize-none py-2 leading-5 [field-sizing:content]",
                  )}
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
      <div ref={footerRef} className="flex h-9 items-center border-t border-grayA-4">
        <button type="button" onClick={draft.add} className={footerButton}>
          <IconPlusOutline18 className="size-3" />
          Add variable
        </button>
        <input
          ref={fileInputRef}
          type="file"
          accept=".env,.txt,text/plain"
          className="hidden"
          onChange={handleFileImport}
        />
        <button
          type="button"
          onClick={() => fileInputRef.current?.click()}
          className={cn(footerButton, "ml-auto")}
        >
          <IconCloudUploadOutline18 className="size-3" />
          Import .env
        </button>
      </div>
    </div>
  );
}
