import type { FormattedParts } from "@/lib/utils/deployment-formatters";

type Props = { label: string; parts: FormattedParts };

export const EnvironmentDisplayValue = ({ label, parts }: Props) => (
  <div className="space-x-1">
    <span className="text-gray-11 text-xs font-normal">{label}</span>
    <span className="font-medium text-gray-12">{parts.value}</span>
    <span className="text-gray-11 font-normal">{parts.unit}</span>
  </div>
);
