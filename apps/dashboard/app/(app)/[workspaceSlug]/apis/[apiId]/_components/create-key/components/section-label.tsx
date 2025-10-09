import { Check, XMark } from "@unkey/icons";
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
            <Check className="text-success-9" iconSize="md-medium" />
          ) : (
            <XMark className="text-error-9" iconSize="md-medium" />
          )}
        </div>
      )}
    </div>
  );
};
