import { cn } from "../../../lib/utils";
import { STATUS_STYLES } from "../constants/constants";

// Row className for DataTables where rows are clickable and one row is
// "selected" (master/detail pattern). Composes the shared STATUS_STYLES so
// every consumer agrees on hover/selected/focus visuals.
export const getSelectableRowClassName = (isSelected: boolean): string =>
  cn(
    STATUS_STYLES.base,
    STATUS_STYLES.hover,
    "group rounded-sm",
    "focus:outline-hidden focus:ring-1 focus:ring-opacity-40",
    STATUS_STYLES.focusRing,
    isSelected && STATUS_STYLES.selected,
  );
