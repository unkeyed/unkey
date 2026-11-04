"use client";

import { PaneSubmit } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/flow/pane-actions";
import type { EnvironmentKind } from "@/lib/queries/environments";
import { IconCircleInfoOutline12, IconPenWriting3Outline18 } from "@unkey/icons";
import {
  Alert,
  AlertDescription,
  Button,
  FormField,
  FormInput,
  InfoTooltip,
  InlineLink,
  RadioCard,
  RadioGroup,
  RadioOption,
} from "@unkey/ui";
import { useState } from "react";
import {
  CLUSTERS,
  type Cluster,
  type Compute,
  ENGINES,
  type Engine,
  type EnvironmentBranches,
  REGIONS,
  clusterCopy,
  engineCopy,
  planetscaleUrls,
  regionLabel,
  sizeSpec,
  skusFor,
} from "../database-model";
import { engineIcon } from "../engine-icon";
import {
  type DatabaseConfig,
  type EnvironmentPlan,
  branchFor,
  clusterAvailable,
  monthlyPrice,
  sourceFlow,
} from "./create-flow-model";
import { EnvironmentHeading, PickerField, Price } from "./fields";
import { PreviewItem } from "./preview-item";
import { ReviewNote, ReviewSummary } from "./review";

const clusterNote: Record<Engine, string | undefined> = {
  postgres: undefined,
  mysql: "MySQL databases on PlanetScale run with high availability.",
};

export function ConfigurePane({
  config,
  nameError,
  canContinue,
  onChange,
  onContinue,
}: {
  config: DatabaseConfig;
  nameError: string | null;
  canContinue: boolean;
  onChange: (config: DatabaseConfig) => void;
  onContinue: () => void;
}) {
  return (
    <>
      <FormInput
        data-1p-ignore
        label="Database name"
        placeholder="orders"
        value={config.name}
        error={nameError ?? undefined}
        onChange={(event) => onChange({ ...config, name: event.target.value })}
      />
      <FormField label="Engine">
        {(control) => (
          <RadioGroup
            id={control.id}
            aria-label="Engine"
            aria-describedby={control.describedBy}
            value={config.engine}
            onValueChange={(engine) => onChange({ ...config, engine })}
            className="sm:grid-cols-2"
          >
            {ENGINES.map((id) => {
              const Icon = engineIcon[id];
              return (
                <RadioCard
                  key={id}
                  value={id}
                  icon={<Icon className="size-4" />}
                  title={engineCopy[id].title}
                  description={engineCopy[id].description}
                />
              );
            })}
          </RadioGroup>
        )}
      </FormField>
      <PickerField
        label="Region"
        value={config.region}
        options={REGIONS.map((region) => ({ value: region, label: regionLabel[region] }))}
        onChange={(region) => onChange({ ...config, region })}
      />
      <PaneSubmit disabled={!canContinue} onClick={onContinue} />
    </>
  );
}

function ClusterField({
  engine,
  value,
  onChange,
}: {
  engine: Engine;
  value: Cluster;
  onChange: (cluster: Cluster) => void;
}) {
  return (
    <FormField label="Cluster" description={clusterNote[engine]}>
      {(control) => (
        <RadioGroup
          id={control.id}
          aria-label="Cluster"
          aria-describedby={control.describedBy}
          value={value}
          onValueChange={onChange}
          className="flex flex-wrap gap-x-5 gap-y-2"
        >
          {CLUSTERS.map((id) => (
            <span key={id} className="inline-flex items-center gap-1.5">
              <RadioOption
                value={id}
                label={clusterCopy[id].label}
                disabled={!clusterAvailable(engine, id)}
              />
              <InfoTooltip
                content={<span className="block max-w-56">{clusterCopy[id].description}</span>}
                position={{ side: "top" }}
                triggerClassName="flex text-gray-9 hover:text-gray-12"
              >
                <IconCircleInfoOutline12 aria-label={`About ${clusterCopy[id].label}`} />
              </InfoTooltip>
            </span>
          ))}
        </RadioGroup>
      )}
    </FormField>
  );
}

function ComputeFields({
  engine,
  compute,
  onChange,
}: {
  engine: Engine;
  compute: Compute;
  onChange: (compute: Compute) => void;
}) {
  const skus = skusFor(engine, compute.cluster);
  return (
    <>
      <ClusterField
        engine={engine}
        value={compute.cluster}
        onChange={(cluster) => onChange({ ...compute, cluster })}
      />
      <PickerField
        label="Size"
        value={compute.size}
        options={skus.map((sku) => ({
          value: sku.id,
          label: `${sku.id} · ${sku.cpu} · ${sku.memory} · $${sku.monthlyPrice}/mo`,
        }))}
        onChange={(size) => onChange({ ...compute, size })}
      />
    </>
  );
}

function PreviewRow({
  engine,
  plan,
  compute,
  onPlan,
  onCompute,
}: {
  engine: Engine;
  plan: EnvironmentPlan;
  compute: Compute;
  onPlan: (plan: EnvironmentPlan) => void;
  onCompute: (compute: Compute) => void;
}) {
  const [editing, setEditing] = useState(false);
  const spec = sizeSpec[compute.size];
  return (
    <PreviewItem
      source="new"
      plan={plan}
      productionBranch={branchFor.production}
      onPlan={onPlan}
      detail={`${clusterCopy[compute.cluster].label} · ${compute.size} · ${spec.cpu} · ${spec.memory}`}
      trailing={
        <>
          <Price value={monthlyPrice(engine, compute)} />
          <Button
            variant="outline"
            size="sm"
            aria-expanded={editing}
            onClick={() => setEditing((open) => !open)}
          >
            {editing ? (
              "Done"
            ) : (
              <>
                <IconPenWriting3Outline18 className="size-3.5" />
                Edit
              </>
            )}
          </Button>
        </>
      }
    >
      {editing ? (
        <div className="flex flex-col gap-4 border-t border-grayA-4 pt-4">
          <ComputeFields engine={engine} compute={compute} onChange={onCompute} />
        </div>
      ) : null}
    </PreviewItem>
  );
}

export function EnvironmentsPane({
  org,
  engine,
  plan,
  compute,
  total,
  hasCard,
  onPlan,
  onCompute,
  onContinue,
}: {
  org: string;
  engine: Engine;
  plan: EnvironmentPlan;
  compute: Record<EnvironmentKind, Compute>;
  total: number;
  hasCard: boolean;
  onPlan: (plan: EnvironmentPlan) => void;
  onCompute: (env: EnvironmentKind, compute: Compute) => void;
  onContinue: () => void;
}) {
  return (
    <>
      <div className="flex flex-col gap-4 rounded-lg border p-4">
        <div className="flex items-center gap-2 text-sm">
          <EnvironmentHeading env="production" />
          <span className="ml-auto">
            <Price value={monthlyPrice(engine, compute.production)} />
          </span>
        </div>
        <ComputeFields
          engine={engine}
          compute={compute.production}
          onChange={(next) => onCompute("production", next)}
        />
      </div>
      <PreviewRow
        engine={engine}
        plan={plan}
        compute={compute.preview}
        onPlan={onPlan}
        onCompute={(next) => onCompute("preview", next)}
      />
      <div className="flex flex-col gap-2 border-t border-grayA-4 pt-4">
        <div className="flex items-center justify-between gap-4 text-sm">
          <span className="font-medium text-gray-12">Monthly total</span>
          <Price value={total} />
        </div>
        <span className="text-xs text-gray-11">Billed by PlanetScale to {org}</span>
      </div>
      {hasCard ? null : (
        <Alert variant="warn">
          <AlertDescription>
            Add a payment method in PlanetScale before creating a database.{" "}
            <InlineLink
              href={planetscaleUrls.billing(org)}
              target="_blank"
              rel="noopener noreferrer"
              label="Open billing"
            />
          </AlertDescription>
        </Alert>
      )}
      <PaneSubmit disabled={!hasCard} onClick={onContinue} />
    </>
  );
}

export function ReviewNewPane({
  org,
  config,
  branches,
  total,
  submitting,
  onSubmit,
}: {
  org: string;
  config: DatabaseConfig;
  branches: EnvironmentBranches;
  total: number;
  submitting: boolean;
  onSubmit: () => void;
}) {
  return (
    <>
      <ReviewSummary
        nameLabel="Database name"
        name={config.name}
        org={org}
        engine={config.engine}
        region={config.region}
        branches={branches}
        price={total}
      />
      <ReviewNote />
      <PaneSubmit loading={submitting} onClick={onSubmit}>
        {sourceFlow.new.submit}
      </PaneSubmit>
    </>
  );
}
