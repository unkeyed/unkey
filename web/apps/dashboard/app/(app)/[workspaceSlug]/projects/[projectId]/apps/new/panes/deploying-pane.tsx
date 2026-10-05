"use client";

import { IconChevronRightOutline12 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import Link from "next/link";
import { useEffect, useState } from "react";
import { CardHeader, cardFooter, cardSurface } from "../card";
import { useNewAppFlow } from "../flow";
import { scrollFadeClass, useScrollFade } from "../use-scroll-fade";
import type { SetupFieldFocus } from "../wizard-model";
import { CongratsBody } from "./deploying/congrats";
import { useCrashHelp } from "./deploying/crash-help";
import { type CrashHelp, directoryLabel, directoryName } from "./deploying/crash-help-state";
import {
  type ResultValue,
  type StageDetail,
  type StageRow,
  isInstanceCrash,
  resultRows,
  stageRows,
  watchFooter,
  watchStatus,
} from "./deploying/deploy-card-state";
import { crashExplanation } from "./deploying/instance-state";
import { LogTicker } from "./deploying/log-ticker";
import type { StageKey } from "./deploying/run-model";
import { LiveUrl, LogBox, stageGlyph } from "./deploying/run-ui";
import { type DeployRun, useDeployRun } from "./deploying/use-deploy-run";

type EditSettings = (focus: SetupFieldFocus | null) => void;

function ResultValueView({ run, value }: { run: DeployRun; value: ResultValue }) {
  return match(value)
    .with({ type: "domain" }, () => <LiveUrl run={run} />)
    .with({ type: "pending" }, ({ text }) => (
      <span className="animate-pulse text-gray-10 motion-reduce:animate-none">{text}</span>
    ))
    .with({ type: "mono" }, ({ text, tone }) => (
      <span
        title={text}
        className={cn("truncate font-mono text-xs", tone === "error" && "text-error-11")}
      >
        {text}
      </span>
    ))
    .exhaustive();
}

function liveRegions(run: DeployRun): string[] {
  const deployment = run.deployment;
  if (!deployment) {
    return [];
  }
  const running = deployment.instances
    .filter((instance) => instance.status === "running")
    .map((instance) => instance.region.name);
  const names = running.length > 0 ? running : deployment.desiredRegions.map((r) => r.region.name);
  return [...new Set(names)];
}

export function Result() {
  const run = useDeployRun();
  const rows = resultRows({
    hasDomain: run.primaryDomain !== null,
    instances: run.instances,
    gitBranch: run.deployment?.gitBranch ?? null,
    gitCommitSha: run.deployment?.gitCommitSha ?? null,
  });
  return (
    <div className={cardSurface}>
      <CongratsBody
        elapsedMs={run.view.elapsedMs}
        regions={liveRegions(run)}
        rows={rows.map((row) => ({
          label: row.label,
          value: <ResultValueView run={run} value={row.value} />,
        }))}
      />
      <div className={cardFooter}>
        <span className="ml-auto flex shrink-0 items-center gap-2">
          <Link href={run.deploymentHref}>
            <Button variant="outline" size="sm">
              View deployment
            </Button>
          </Link>
          <Link href={run.appHref}>
            <Button variant="primary" size="sm">
              Go to app
            </Button>
          </Link>
        </span>
      </div>
    </div>
  );
}

export function Watch({ appId }: { appId: string }) {
  const { projectId, dispatch } = useNewAppFlow();
  const run = useDeployRun();
  const help = useCrashHelp(projectId, appId);
  const rows = stageRows(run.view, isInstanceCrash(run.view, run.instances));
  const onEditSettings: EditSettings = (focus) => dispatch({ type: "edit-settings", focus });
  const onNext = () => dispatch({ type: "go", card: "result" });
  const scrollRef = useScrollFade();
  const live = run.view.outcome === "live";
  useEffect(() => {
    if (live) {
      dispatch({ type: "go", card: "result" });
    }
  }, [live, dispatch]);
  return (
    <div className={cardSurface}>
      <div
        ref={scrollRef}
        className={cn(
          "flex min-h-0 flex-col gap-4 overflow-y-auto overscroll-contain p-5 [scrollbar-width:thin]",
          scrollFadeClass,
        )}
      >
        <CardHeader
          title="Deployment"
          description={watchStatus[run.view.outcome]}
          elapsedMs={run.view.elapsedMs}
        />
        <StageAccordion
          run={run}
          rows={rows}
          help={help}
          failedKey={run.view.failedStage?.key ?? null}
          onEditSettings={onEditSettings}
        />
      </div>
      <WatchActions run={run} onEditSettings={onEditSettings} onNext={onNext} />
    </div>
  );
}

function WatchActions({
  run,
  onEditSettings,
  onNext,
}: { run: DeployRun; onEditSettings: EditSettings; onNext: () => void }) {
  return match(watchFooter[run.view.outcome])
    .with("none", () => null)
    .with("continue", () => (
      <div className={cn(cardFooter, "justify-end")}>
        <Button variant="primary" size="sm" onClick={onNext}>
          Continue
        </Button>
      </div>
    ))
    .with("blocked", () => (
      <div className={cn(cardFooter, "justify-end")}>
        <Link href={run.deploymentHref}>
          <Button variant="primary" size="sm">
            View deployment
          </Button>
        </Link>
      </div>
    ))
    .with("failed", () => (
      <div className={cardFooter}>
        <span className="ml-auto flex shrink-0 items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => onEditSettings(null)}>
            Edit settings
          </Button>
          <Link href={run.deploymentHref}>
            <Button variant="outline" size="sm">
              View deployment
            </Button>
          </Link>
          <Button
            variant="primary"
            size="sm"
            loading={run.retry.isDeploying}
            disabled={!run.retry.canDeploy}
            onClick={run.retry.start}
          >
            Redeploy
          </Button>
        </span>
      </div>
    ))
    .exhaustive();
}

function StageAccordion({
  run,
  rows,
  help,
  failedKey,
  onEditSettings,
}: {
  run: DeployRun;
  rows: StageRow[];
  help: CrashHelp;
  failedKey: StageKey | null;
  onEditSettings: EditSettings;
}) {
  const [openKey, setOpenKey] = useState<StageKey | null>(failedKey);
  const [seenFailedKey, setSeenFailedKey] = useState(failedKey);
  if (failedKey !== seenFailedKey) {
    setSeenFailedKey(failedKey);
    if (failedKey) {
      setOpenKey(failedKey);
    }
  }
  return (
    <div className="flex min-h-0 flex-col overflow-hidden rounded-lg border border-grayA-4">
      {rows.map((row) => {
        const open = row.expandable && openKey === row.key;
        return (
          <div
            key={row.key}
            className={cn(
              "border-t border-grayA-4 first:border-t-0",
              open && row.fillsCard && "flex min-h-0 flex-col",
            )}
          >
            <button
              type="button"
              disabled={!row.expandable}
              onClick={() => setOpenKey((key) => (key === row.key ? null : row.key))}
              className="flex h-11 w-full shrink-0 items-center gap-3 px-4 text-left disabled:cursor-default enabled:hover:bg-grayA-2"
            >
              <IconChevronRightOutline12
                className={cn(
                  "size-3 shrink-0 text-gray-9 transition-transform duration-200 ease-out motion-reduce:transition-none",
                  open && "rotate-90",
                  !row.expandable && "opacity-0",
                )}
              />
              <span className="shrink-0 text-sm font-medium text-gray-12">{row.title}</span>
              <span className="min-w-0 flex-1">
                {open ? null : <LogTicker run={run} stage={row.stage} />}
              </span>
              <span className="font-mono text-xs tabular-nums text-gray-10">{row.meta}</span>
              <span className="grid size-4 shrink-0 place-items-center">
                {stageGlyph[row.stage.state]}
              </span>
            </button>
            {open ? (
              <StageDetailView
                run={run}
                detail={row.detail}
                help={help}
                onEditSettings={onEditSettings}
              />
            ) : null}
          </div>
        );
      })}
    </div>
  );
}

function StageDetailView({
  run,
  detail,
  help,
  onEditSettings,
}: {
  run: DeployRun;
  detail: StageDetail;
  help: CrashHelp;
  onEditSettings: EditSettings;
}) {
  return match(detail)
    .with({ type: "none" }, () => null)
    .with({ type: "logs" }, ({ error }) => (
      <div className="flex min-h-0 flex-col overflow-hidden border-t border-grayA-4">
        {error ? (
          <div className="shrink-0 border-b border-grayA-4 bg-error-2 px-4 py-2.5 font-mono text-xs leading-5 text-error-11">
            {error}
          </div>
        ) : null}
        <LogBox run={run} className="min-h-0 flex-1 rounded-none border-0" />
      </div>
    ))
    .with({ type: "crash" }, () => (
      <CrashDetail run={run} help={help} onEditSettings={onEditSettings} />
    ))
    .with({ type: "error" }, ({ message }) => (
      <div className="border-t border-grayA-4 bg-error-2 px-4 py-3 font-mono text-xs leading-5 text-error-11">
        {message}
      </div>
    ))
    .exhaustive();
}

const RUNTIME_TAIL_LINES = 8;

function CrashDetail({
  run,
  help,
  onEditSettings,
}: { run: DeployRun; help: CrashHelp; onEditSettings: EditSettings }) {
  const lines =
    run.groups.find((group) => group.id === "runtime")?.lines.slice(-RUNTIME_TAIL_LINES) ?? [];
  const instances = run.instances;
  return (
    <div className="flex flex-col gap-2 border-t border-grayA-4 bg-error-2 px-4 py-3 text-xs leading-5">
      <p className="text-error-11">
        {crashExplanation(run.deployment?.lastExit ?? null, lines.length > 0)}
      </p>
      {instances?.lastExit ? <p className="font-mono text-gray-10">{instances.lastExit}</p> : null}
      <CrashHelpView help={help} onEditSettings={onEditSettings} />
      {lines.length > 0 ? (
        <div className="font-mono text-gray-12">
          {lines.map((line) => (
            <div key={line.id} className="whitespace-pre-wrap break-all">
              {line.text}
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function CrashHelpView({
  help,
  onEditSettings,
}: { help: CrashHelp; onEditSettings: EditSettings }) {
  return match(help)
    .with({ type: "loading" }, () => null)
    .with({ type: "dockerfile" }, ({ dockerfile, rootDirectory }) => (
      <p className="text-gray-11">
        This app built from {directoryName(rootDirectory)} without a Dockerfile, but the repository
        has one at <span className="font-mono text-gray-12">{dockerfile}</span>.{" "}
        <button
          type="button"
          onClick={() => onEditSettings("dockerfile")}
          className="font-medium text-gray-12 underline decoration-grayA-6 underline-offset-2 hover:decoration-gray-12"
        >
          Use this Dockerfile
        </button>
      </p>
    ))
    .with({ type: "checklist" }, ({ port, startCommand, rootDirectory }) => (
      <ul className="flex flex-col text-gray-11">
        <li>
          Your app must keep running and listen on port{" "}
          <span className="font-mono text-gray-12">{port}</span>.
        </li>
        <li>
          Start command: <span className="font-mono text-gray-12">{startCommand}</span>
        </li>
        <li>
          Root directory:{" "}
          <span className="font-mono text-gray-12">{directoryLabel(rootDirectory)}</span>
        </li>
      </ul>
    ))
    .exhaustive();
}
