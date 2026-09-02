import { IconCheckOutline18, IconXmarkOutline18 } from "nucleo-ui-outline-18";
import type { SectionState } from "../types";

export const SectionLabel = ({
  label,
  validState,
}: {
  label: string;
  validState: SectionState;
}) => {
  return (
    <div className="w-full justify-between flex items-center">
      {label}
      {validState !== "initial" && (
        <div className="ml-auto">
          {validState === "valid" ? (
            <IconCheckOutline18 className="size-4 text-success-9" />
          ) : (
            <IconXmarkOutline18 className="size-4 text-error-9" />
          )}
        </div>
      )}
    </div>
  );
};
