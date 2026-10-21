import { type Environment, productionFirst } from "@/lib/collections/deploy/environments";
import { IconChevronDownOutline18 } from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import type { ComponentProps } from "react";
import { useProjectData } from "../[appId]/(overview)/data-provider";
import { EnvironmentLabel, environmentName } from "./environment-label";

import { ALL_ENVIRONMENTS } from "./environment-scope";

type Option = { value: string; label: string; environment?: Environment };

type EnvironmentSelectProps = {
  value: string;
  onValueChange: (value: string) => void;
  includeAll?: boolean;
  placeholder?: string;
  disabled?: boolean;
} & Pick<
  ComponentProps<typeof SelectTrigger>,
  "id" | "className" | "wrapperClassName" | "leftIcon" | "variant"
>;

export function EnvironmentSelect({
  value,
  onValueChange,
  includeAll = true,
  placeholder,
  disabled,
  ...triggerProps
}: EnvironmentSelectProps) {
  const { environments } = useProjectData();
  const options: Option[] = [
    ...(includeAll ? [{ value: ALL_ENVIRONMENTS, label: "All environments" }] : []),
    ...productionFirst(environments).map((environment) => ({
      value: environment.id,
      label: environmentName(environment.slug),
      environment,
    })),
  ];
  const selected =
    options.find((option) => option.value === value) ?? (includeAll ? options[0] : undefined);

  return (
    <Select
      value={selected?.value ?? null}
      items={options}
      disabled={disabled}
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
        <SelectValue placeholder={placeholder}>
          {selected ? <OptionLabel option={selected} /> : undefined}
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
