"use client";

import { BrandLogo } from "@/lib/extensions/brand-logos";
import { useInstallations } from "@/lib/extensions/installations";
import {
  type Extension,
  type ExtensionConfigState,
  getExtensionBySlug,
} from "@/lib/extensions/registry";
import { Check, CircleCheck, CircleWarning, TriangleWarning2 } from "@unkey/icons";
import { Button, FormInput, toast } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import Link from "next/link";
import { notFound, useParams, useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { ProjectContentWrapper } from "../../../../components/project-content-wrapper";
import { ConfigForm, initialConfigState, validateConfigState } from "../../components/config-form";

type WizardStepId = "permissions" | "configuration" | "oauth" | "verify" | "done";

type FormState = {
  instanceName: string;
  config: ExtensionConfigState;
  oauthConnected: boolean;
};

export default function InstallExtensionPage() {
  const params = useParams<{
    workspaceSlug: string;
    projectId: string;
    extensionSlug: string;
  }>();
  const router = useRouter();

  const extension = getExtensionBySlug(params.extensionSlug);
  if (!extension) {
    notFound();
  }

  const basePath = `/${params.workspaceSlug}/projects/${params.projectId}/extensions`;
  const detailPath = `${basePath}/${extension.slug}`;
  const installedPath = `${basePath}/installed`;

  const { create, installations } = useInstallations(params.projectId);

  const defaultInstanceName = useMemo(
    () =>
      suggestInstanceName(
        extension,
        installations.map((i) => i.instanceName),
      ),
    [extension, installations],
  );

  const [form, setForm] = useState<FormState>(() => ({
    instanceName: defaultInstanceName,
    config: initialConfigState(extension.configFields),
    oauthConnected: false,
  }));

  const steps: WizardStepId[] = useMemo(() => {
    const base: WizardStepId[] = ["permissions", "configuration"];
    if (extension.oauth) {
      base.push("oauth");
    }
    base.push("verify", "done");
    return base;
  }, [extension.oauth]);

  const [stepId, setStepId] = useState<WizardStepId>("permissions");
  const stepIndex = steps.indexOf(stepId);

  const goNext = () => {
    const next = steps[stepIndex + 1];
    if (next) {
      setStepId(next);
    }
  };
  const goBack = () => {
    const prev = steps[stepIndex - 1];
    if (prev) {
      setStepId(prev);
    }
  };

  const handleComplete = async (_results: Array<{ id: string; ok: boolean }>) => {
    try {
      await create({
        extensionSlug: extension.slug,
        instanceName: form.instanceName.trim() || extension.name,
        config: form.config,
        oauthConnected: form.oauthConnected,
      });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to install extension");
      return;
    }
    // Continue to the "done" step regardless of verification pass/fail; the
    // results are already shown in the verify step itself.
    goNext();
  };

  return (
    <ProjectContentWrapper centered maxWidth="780px" className="mt-8">
      <Link href={detailPath} className="text-[12px] text-grayA-10 hover:text-grayA-12 w-fit">
        ← Back to {extension.name}
      </Link>

      <Header extension={extension} />
      <Stepper steps={steps} activeIndex={stepIndex} />

      <div className="rounded-lg border border-grayA-3 p-6">
        {stepId === "permissions" && (
          <PermissionsStep
            extension={extension}
            instanceName={form.instanceName}
            onInstanceNameChange={(v) => setForm((f) => ({ ...f, instanceName: v }))}
            onNext={goNext}
            onCancel={() => router.push(detailPath)}
          />
        )}
        {stepId === "configuration" && (
          <ConfigurationStep
            extension={extension}
            values={form.config}
            onChange={(config) => setForm((f) => ({ ...f, config }))}
            onBack={goBack}
            onNext={goNext}
          />
        )}
        {stepId === "oauth" && extension.oauth && (
          <OAuthStep
            extension={extension}
            connected={form.oauthConnected}
            onConnect={() => setForm((f) => ({ ...f, oauthConnected: true }))}
            onBack={goBack}
            onNext={goNext}
          />
        )}
        {stepId === "verify" && (
          <VerifyStep extension={extension} onBack={goBack} onComplete={handleComplete} />
        )}
        {stepId === "done" && (
          <DoneStep
            extension={extension}
            instanceName={form.instanceName}
            installedPath={installedPath}
            backPath={basePath}
          />
        )}
      </div>
    </ProjectContentWrapper>
  );
}

function Header({ extension }: { extension: Extension }) {
  return (
    <div className="flex items-center gap-3">
      <div className="flex h-10 w-10 items-center justify-center rounded-[10px] bg-white shadow-sm shadow-grayA-8/20 ring-1 ring-grayA-3 overflow-hidden shrink-0">
        <BrandLogo
          slug={extension.slug}
          iconUrl={extension.iconUrl}
          name={extension.name}
          className="size-6"
        />
      </div>
      <div className="flex flex-col">
        <h1 className="text-[18px] font-semibold text-grayA-12">Install {extension.name}</h1>
        <p className="text-[12px] text-grayA-10">{extension.tagline}</p>
      </div>
    </div>
  );
}

function Stepper({ steps, activeIndex }: { steps: WizardStepId[]; activeIndex: number }) {
  return (
    <ol className="flex items-center gap-2">
      {steps.map((step, i) => (
        <li key={step} className="flex items-center gap-2">
          <span
            className={cn(
              "flex h-6 w-6 items-center justify-center rounded-full text-[11px] font-medium border",
              i < activeIndex
                ? "bg-successA-3 text-successA-11 border-successA-6"
                : i === activeIndex
                  ? "bg-grayA-12 text-gray-1 border-grayA-12"
                  : "bg-background text-grayA-10 border-grayA-4",
            )}
          >
            {i < activeIndex ? <Check className="size-3" /> : i + 1}
          </span>
          <span
            className={cn(
              "text-[12px]",
              i === activeIndex ? "text-grayA-12 font-medium" : "text-grayA-10",
            )}
          >
            {STEP_LABELS[step]}
          </span>
          {i < steps.length - 1 ? <span className="text-grayA-6">·</span> : null}
        </li>
      ))}
    </ol>
  );
}

const STEP_LABELS: Record<WizardStepId, string> = {
  permissions: "Permissions",
  configuration: "Configuration",
  oauth: "Connect",
  verify: "Verify",
  done: "Done",
};

function PermissionsStep({
  extension,
  instanceName,
  onInstanceNameChange,
  onNext,
  onCancel,
}: {
  extension: Extension;
  instanceName: string;
  onInstanceNameChange: (value: string) => void;
  onNext: () => void;
  onCancel: () => void;
}) {
  const trimmed = instanceName.trim();
  const canContinue = trimmed.length > 0;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-2">
        <h2 className="text-[14px] font-semibold text-grayA-12">Name this installation</h2>
        <p className="text-[12px] text-grayA-10">
          You can install {extension.name} multiple times — give this one a name so you can tell
          them apart.
        </p>
        <FormInput
          value={instanceName}
          onChange={(e) => onInstanceNameChange(e.target.value)}
          placeholder={`${extension.name} (Production)`}
        />
      </div>

      <div className="flex flex-col gap-2">
        <h2 className="text-[14px] font-semibold text-grayA-12">{extension.name} will access:</h2>
        <ul className="flex flex-col gap-2">
          {extension.permissions.map((permission) => (
            <li
              key={permission.id}
              className="flex flex-col gap-0.5 rounded-md border border-grayA-3 p-3"
            >
              <span className="text-[13px] font-medium text-grayA-12">{permission.label}</span>
              <span className="text-[12px] text-grayA-11">{permission.description}</span>
            </li>
          ))}
        </ul>
      </div>

      {extension.scope === "workspace" ? (
        <div className="flex items-start gap-2 rounded-md bg-warningA-2 border border-warningA-4 p-3 text-[12px] text-warningA-11">
          <TriangleWarning2 className="size-4 shrink-0 mt-0.5" />
          <span>
            This extension installs at the <strong>workspace</strong> level and applies to all
            projects in this workspace.
          </span>
        </div>
      ) : null}

      <Footer onBack={onCancel} backLabel="Cancel" onNext={onNext} canContinue={canContinue} />
    </div>
  );
}

function ConfigurationStep({
  extension,
  values,
  onChange,
  onBack,
  onNext,
}: {
  extension: Extension;
  values: ExtensionConfigState;
  onChange: (values: ExtensionConfigState) => void;
  onBack: () => void;
  onNext: () => void;
}) {
  const errors = validateConfigState(extension.configFields, values);
  const canContinue = errors.length === 0;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h2 className="text-[14px] font-semibold text-grayA-12">Configuration</h2>
        <p className="text-[12px] text-grayA-10">
          Provide the credentials and settings {extension.name} needs to run.
        </p>
      </div>

      <ConfigForm fields={extension.configFields} values={values} onChange={onChange} />

      <Footer onBack={onBack} onNext={onNext} canContinue={canContinue} />
    </div>
  );
}

function OAuthStep({
  extension,
  connected,
  onConnect,
  onBack,
  onNext,
}: {
  extension: Extension;
  connected: boolean;
  onConnect: () => void;
  onBack: () => void;
  onNext: () => void;
}) {
  const [pending, setPending] = useState(false);
  const oauth = extension.oauth;
  if (!oauth) {
    return null;
  }

  const handleConnect = async () => {
    setPending(true);
    // Stubbed OAuth flow — a real implementation redirects to provider.
    await new Promise((resolve) => setTimeout(resolve, 800));
    onConnect();
    setPending(false);
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h2 className="text-[14px] font-semibold text-grayA-12">Connect with {oauth.provider}</h2>
        <p className="text-[12px] text-grayA-10">
          Authorize Unkey to act on your behalf in {oauth.provider}.
        </p>
      </div>

      <div className="flex flex-col gap-2 rounded-md border border-grayA-3 p-3">
        <span className="text-[11px] uppercase tracking-wide text-grayA-10">Requested scopes</span>
        <ul className="flex flex-col gap-1">
          {oauth.scopes.map((scope) => (
            <li key={scope} className="text-[12px] text-grayA-11 font-mono">
              · {scope}
            </li>
          ))}
        </ul>
      </div>

      {connected ? (
        <div className="flex items-center gap-2 rounded-md bg-successA-2 border border-successA-4 p-3 text-[12px] text-successA-11">
          <CircleCheck className="size-4" />
          <span>Connected to {oauth.provider}.</span>
        </div>
      ) : (
        <Button onClick={handleConnect} loading={pending} className="w-fit">
          Connect with {oauth.provider}
        </Button>
      )}

      <Footer onBack={onBack} onNext={onNext} canContinue={connected} />
    </div>
  );
}

type CheckResult = { id: string; ok: boolean; message?: string };

function VerifyStep({
  extension,
  onBack,
  onComplete,
}: {
  extension: Extension;
  onBack: () => void;
  onComplete: (results: CheckResult[]) => void;
}) {
  const [results, setResults] = useState<Map<string, CheckResult>>(new Map());
  const [running, setRunning] = useState(false);
  const [done, setDone] = useState(false);

  const run = async () => {
    setRunning(true);
    setResults(new Map());
    setDone(false);

    const next = new Map<string, CheckResult>();
    for (const check of extension.verification) {
      // Stubbed: each check resolves to ok after a short delay.
      // Replace with `check.run(config)` when manifests provide it.
      await new Promise((resolve) => setTimeout(resolve, 500));
      const result: CheckResult = { id: check.id, ok: true };
      next.set(check.id, result);
      setResults(new Map(next));
    }

    setRunning(false);
    setDone(true);
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h2 className="text-[14px] font-semibold text-grayA-12">Verify</h2>
        <p className="text-[12px] text-grayA-10">
          Run the extension's pre-flight checks against your configuration before installing.
        </p>
      </div>

      <ul className="flex flex-col gap-2">
        {extension.verification.map((check) => {
          const result = results.get(check.id);
          return (
            <li
              key={check.id}
              className="flex items-center gap-3 rounded-md border border-grayA-3 p-3"
            >
              <CheckIndicator state={result ? (result.ok ? "ok" : "fail") : "idle"} />
              <span className="text-[13px] text-grayA-12 flex-1">{check.label}</span>
              {result?.message ? (
                <span className="text-[11px] text-grayA-10">{result.message}</span>
              ) : null}
            </li>
          );
        })}
      </ul>

      <div className="flex items-center justify-between gap-3">
        <Button variant="outline" onClick={onBack} disabled={running}>
          Back
        </Button>
        <div className="flex gap-2">
          {done ? (
            <Button onClick={() => onComplete([...results.values()])}>Install</Button>
          ) : (
            <Button onClick={run} loading={running}>
              Run checks
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}

function DoneStep({
  extension,
  instanceName,
  installedPath,
  backPath,
}: {
  extension: Extension;
  instanceName: string;
  installedPath: string;
  backPath: string;
}) {
  return (
    <div className="flex flex-col items-center gap-4 py-8 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-successA-3 text-successA-11">
        <CircleCheck className="size-6" />
      </div>
      <div className="flex flex-col gap-1">
        <h2 className="text-[16px] font-semibold text-grayA-12">{extension.name} installed</h2>
        <p className="text-[13px] text-grayA-10">
          "{instanceName}" is now active for this project.
        </p>
      </div>
      <div className="flex gap-2">
        <Button variant="outline" asChild>
          <Link href={backPath}>Back to marketplace</Link>
        </Button>
        <Button asChild>
          <Link href={installedPath}>View installations</Link>
        </Button>
      </div>
    </div>
  );
}

function CheckIndicator({ state }: { state: "idle" | "ok" | "fail" }) {
  if (state === "idle") {
    return <span className="block size-4 rounded-full border border-grayA-4" />;
  }
  if (state === "ok") {
    return <CircleCheck className="size-4 text-successA-11" />;
  }
  return <CircleWarning className="size-4 text-errorA-11" />;
}

function Footer({
  onBack,
  backLabel = "Back",
  onNext,
  canContinue,
}: {
  onBack: () => void;
  backLabel?: string;
  onNext: () => void;
  canContinue: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <Button variant="outline" onClick={onBack}>
        {backLabel}
      </Button>
      <Button onClick={onNext} disabled={!canContinue}>
        Continue
      </Button>
    </div>
  );
}

function suggestInstanceName(extension: Extension, existing: string[]): string {
  const base = extension.name;
  if (!existing.includes(base)) {
    return base;
  }
  let n = 2;
  while (existing.includes(`${base} ${n}`)) {
    n++;
  }
  return `${base} ${n}`;
}
