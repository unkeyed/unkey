import { Label } from "@/components/ui/label";
import { CircleInfo } from "@unkey/icons";
import type { ReactNode } from "react";
import { InputTooltip } from "./input-tooltip";

type FormFieldProps = {
  label: string;
  tooltip?: string;
  error?: string;
  children: ReactNode;
};

export const FormField = ({ label, tooltip, error, children }: FormFieldProps) => (
  // biome-ignore lint/a11y/useKeyWithClickEvents: no need for button
  <div className="flex flex-col gap-1" onClick={(e) => e.stopPropagation()}>
    <Label
      className="text-gray-11 text-[13px] flex items-center"
      onClick={(e) => e.preventDefault()}
    >
      {label}
      {tooltip && (
        <InputTooltip desc={tooltip}>
          <CircleInfo size="md-regular" className="text-accent-8 ml-[10px]" />
        </InputTooltip>
      )}
    </Label>
    {children}
    {error && <span className="text-error-10 text-[13px] font-medium">{error}</span>}
  </div>
);
