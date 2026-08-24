"use client";

import { instanceLabels, useScopeInstances } from "../hooks/use-scope-instances";
import { type Policy, policySummary } from "../lib/policy";
import { DataList, DataRow } from "./data-row";

type ReviewStageProps = {
  name: string;
  policies: Policy[];
};

export function ReviewStage({ name, policies }: ReviewStageProps) {
  return (
    <DataList>
      <DataRow label="Name">{name}</DataRow>
      {policies.map((policy, index) => (
        <PolicyRow
          key={`${index}-${policy.scope}`}
          label={index === 0 ? "Permissions" : ""}
          policy={policy}
        />
      ))}
    </DataList>
  );
}

type PolicyRowProps = {
  label: string;
  policy: Policy;
};

function PolicyRow({ label, policy }: PolicyRowProps) {
  const { instances } = useScopeInstances(policy.scope);
  const summary = policySummary(policy, instanceLabels(instances));

  return (
    <DataRow label={label}>
      <div className="flex flex-col gap-1">
        <span>{summary.scopeLine}</span>
        {summary.grants.map((grant) => (
          <span key={grant} className="font-normal text-gray-11">
            {grant}
          </span>
        ))}
      </div>
    </DataRow>
  );
}
