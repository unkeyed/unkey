// When editingIndex is NO_ITEM_EDITING (-1), no item is in edit mode.
export const NO_ITEM_EDITING = -1;
export const KEYS = {
  ESCAPE: "Escape",
  ENTER: "Enter",
} as const;
export const PLACEHOLDER_TEXT = "Search logs...";

export const OPTIONS = [
  { value: "requestId", label: "requestId: " },
  { value: "host", label: "host: " },
  { value: "method", label: "method: " },
  { value: "path", label: "path: " },
] as const;

export const OPTION_EXPLANATIONS: Record<string, string> = {
  requestId: "Request identifier",
  host: "Domain name",
  method: "HTTP method",
  path: "Request URL path",
};
