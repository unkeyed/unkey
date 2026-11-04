import { useCallback, useState } from "react";

const NO_SELECTION: Set<string> = new Set();

export function useViewSelection(viewKey: string) {
  const [selection, setSelection] = useState<{ viewKey: string; ids: Set<string> }>({
    viewKey,
    ids: NO_SELECTION,
  });
  if (selection.viewKey !== viewKey) {
    setSelection({ viewKey, ids: NO_SELECTION });
  }

  const setSelected = useCallback((ids: Set<string>) => setSelection({ viewKey, ids }), [viewKey]);

  const toggle = useCallback(
    (id: string) =>
      setSelection((prev) => {
        const ids = new Set(prev.viewKey === viewKey ? prev.ids : NO_SELECTION);
        if (!ids.delete(id)) {
          ids.add(id);
        }
        return { viewKey, ids };
      }),
    [viewKey],
  );

  return {
    selected: selection.viewKey === viewKey ? selection.ids : NO_SELECTION,
    setSelected,
    toggle,
  };
}
