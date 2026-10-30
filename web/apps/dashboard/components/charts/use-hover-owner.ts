import { type Dispatch, type SetStateAction, useEffect, useId } from "react";

export type ActiveMarker = { id: string; owner: string } | null;

export type ActiveChange = Dispatch<SetStateAction<ActiveMarker>>;

export function useHoverOwner(
  active: ActiveMarker,
  onActiveChange: ActiveChange,
  keys: ReadonlyArray<string>,
) {
  const overlay = useId();
  const prefix = `${overlay}:`;
  const ownedKey = active?.owner.startsWith(prefix) ? active.owner.slice(prefix.length) : null;
  const orphaned = ownedKey !== null && !keys.includes(ownedKey) ? active?.owner : undefined;

  useEffect(() => {
    if (orphaned) {
      onActiveChange((current) => (current?.owner === orphaned ? null : current));
    }
  }, [orphaned, onActiveChange]);

  useEffect(
    () => () => onActiveChange((current) => (current?.owner.startsWith(prefix) ? null : current)),
    [prefix, onActiveChange],
  );

  return (key: string, id: string) => {
    const owner = `${prefix}${key}`;
    return {
      open: active?.owner === owner,
      onOpenChange: (open: boolean) => {
        if (open) {
          onActiveChange({ id, owner });
          return;
        }
        onActiveChange((current) => (current?.owner === owner ? null : current));
      },
    };
  };
}
