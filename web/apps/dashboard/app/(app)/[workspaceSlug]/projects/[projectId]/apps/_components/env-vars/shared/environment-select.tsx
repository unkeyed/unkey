import type { Environment } from "@/lib/collections/deploy/environments";
import { IconChevronDownOutline18 } from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import type { ComponentProps } from "react";
import { useProjectData } from "../../../[appId]/(overview)/data-provider";
import { EnvironmentLabel } from "../../environment-label";

export const ALL_ENVIRONMENTS = "all";

type Option = { value: string; label: string; environment?: Environment };

type EnvironmentSelectProps = {
  value: string;
  onValueChange: (value: string) => void;
} & Pick<
  ComponentProps<typeof SelectTrigger>,
  "id" | "className" | "wrapperClassName" | "leftIcon"
>;

export function EnvironmentSelect({
  value,
  onValueChange,
  ...triggerProps
}: EnvironmentSelectProps) {
  const { environments } = useProjectData();
  const options: Option[] = [
    { value: ALL_ENVIRONMENTS, label: "All environments" },
    ...environments.map((environment) => ({
      value: environment.id,
      label: environment.slug,
      environment,
    })),
  ];
  const selected = options.find((option) => option.value === value) ?? options[0];

  return (
    <Select
      value={selected.value}
      items={options}
      onValueChange={(next) => {
        if (next !== null) {
          onValueChange(next);
        }
      }}
    >
      <SelectTrigger
        {...triggerProps}
        rightIcon={<IconChevronDownOutline18 className="size-3.5 absolute right-2" />}
      >
        <SelectValue>
          <OptionLabel option={selected} />
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            <OptionLabel option={option} />
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function OptionLabel({ option }: { option: Option }) {
  return option.environment ? (
    <EnvironmentLabel environment={option.environment} className="text-sm text-gray-12" />
  ) : (
    option.label
  );
}
