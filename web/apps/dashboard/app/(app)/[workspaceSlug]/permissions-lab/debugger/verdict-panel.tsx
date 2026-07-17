"use client";

/**
 * Renders the result of one access check: a large allow/deny verdict, the
 * grants that produced it, and for denials a "why not" analysis ordered by
 * how close each grant came to matching.
 */

import { Badge, Button } from "@unkey/ui";
import { UrnText } from "../components/urn-display";
import { type Analysis, type AnalyzedGrant, sourceText } from "./analysis";

export function VerdictPanel({
  analysis,
  principalName,
  resourceLabel,
  action,
  onRequestGrant,
}: {
  analysis: Analysis;
  principalName: string;
  resourceLabel: string;
  action: string;
  onRequestGrant: (permission: string) => void;
}) {
  return analysis.allowed ? (
    <AllowedVerdict
      analysis={analysis}
      principalName={principalName}
      resourceLabel={resourceLabel}
      action={action}
    />
  ) : (
    <DeniedVerdict
      analysis={analysis}
      principalName={principalName}
      resourceLabel={resourceLabel}
      action={action}
      onRequestGrant={onRequestGrant}
    />
  );
}

function AllowedVerdict({
  analysis,
  principalName,
  resourceLabel,
  action,
}: {
  analysis: Analysis;
  principalName: string;
  resourceLabel: string;
  action: string;
}) {
  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-3 rounded-lg border border-success-6 bg-successA-2 p-5">
        <div className="flex flex-wrap items-center gap-3">
          <span className="text-2xl font-semibold uppercase tracking-wider text-success-11">
            Allowed
          </span>
          <Badge variant="success">
            {analysis.matched.length === 1
              ? "1 matching grant"
              : `${analysis.matched.length} matching grants`}
          </Badge>
        </div>
        <div className="overflow-x-auto">
          <UrnText value={analysis.requestString} />
        </div>
        <p className="text-sm text-gray-11">
          {principalName} can <span className="font-mono text-xs text-info-11">{action}</span> on{" "}
          {resourceLabel}.
        </p>
      </div>

      <section className="flex flex-col gap-2">
        <h3 className="text-[11px] uppercase tracking-wide text-gray-9">Matched by</h3>
        <ul className="flex flex-col gap-2">
          {analysis.matched.map((grant) => (
            <li
              key={grant.permission}
              className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-lg border border-success-6 bg-successA-2 px-3 py-2"
            >
              <div className="overflow-x-auto">
                <UrnText value={grant.permission} />
              </div>
              <Badge variant="success" size="sm">
                matched
              </Badge>
              <span className="ml-auto text-[11px] text-gray-10">{sourceText(grant.sources)}</span>
            </li>
          ))}
        </ul>
      </section>

      {analysis.others.length > 0 && (
        <CollapsedGrantList
          summary={
            analysis.others.length === 1
              ? "Everything else: 1 grant that did not match"
              : `Everything else: ${analysis.others.length} grants that did not match`
          }
          grants={analysis.others}
        />
      )}
    </div>
  );
}

function DeniedVerdict({
  analysis,
  principalName,
  resourceLabel,
  action,
  onRequestGrant,
}: {
  analysis: Analysis;
  principalName: string;
  resourceLabel: string;
  action: string;
  onRequestGrant: (permission: string) => void;
}) {
  const totalGrants = analysis.matched.length + analysis.others.length;

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-3 rounded-lg border border-error-6 bg-errorA-2 p-5">
        <div className="flex flex-wrap items-center gap-3">
          <span className="text-2xl font-semibold uppercase tracking-wider text-error-11">
            Denied
          </span>
          <Badge variant="error">no grant covers this request</Badge>
        </div>
        <div className="overflow-x-auto">
          <UrnText value={analysis.requestString} />
        </div>
        <p className="text-sm text-gray-11">
          {principalName} cannot <span className="font-mono text-xs text-info-11">{action}</span> on{" "}
          {resourceLabel}.
        </p>
        <div className="flex flex-wrap items-center gap-3 pt-1">
          <Button variant="outline" onClick={() => onRequestGrant(analysis.requestString)}>
            Grant exactly this
          </Button>
          <span className="text-xs text-gray-10">
            Adds a direct grant for this one resource and action, nothing broader.
          </span>
        </div>
      </div>

      {totalGrants === 0 ? (
        <p className="rounded-lg border border-grayA-4 bg-grayA-2 px-4 py-6 text-sm text-gray-10">
          {principalName} has no effective grants at all, so nothing could match. Grant the exact
          permission above, or attach a role first.
        </p>
      ) : (
        <div className="flex flex-col gap-5">
          {analysis.wrongAction.length > 0 && (
            <section className="flex flex-col gap-2">
              <header className="flex items-baseline gap-2">
                <h3 className="text-[11px] uppercase tracking-wide text-gray-9">
                  Right resource, wrong action
                </h3>
                <span className="text-[11px] text-gray-9">
                  these grants reach the resource but allow something else
                </span>
              </header>
              <ul className="flex flex-col gap-2">
                {analysis.wrongAction.map((grant) => (
                  <li
                    key={grant.permission}
                    className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-lg border border-grayA-4 px-3 py-2"
                  >
                    <div className="overflow-x-auto">
                      <UrnText value={grant.permission} />
                    </div>
                    <Badge variant="warning" size="sm" font="mono">
                      grants {grant.grantAction}
                    </Badge>
                    <span className="ml-auto text-[11px] text-gray-10">
                      {sourceText(grant.sources)}
                    </span>
                  </li>
                ))}
              </ul>
            </section>
          )}

          {analysis.tooShallow.length > 0 && (
            <section className="flex flex-col gap-2">
              <header className="flex items-baseline gap-2">
                <h3 className="text-[11px] uppercase tracking-wide text-gray-9">
                  Almost reaches it
                </h3>
                <span className="text-[11px] text-gray-9">
                  these patterns stop one level above the resource
                </span>
              </header>
              <ul className="flex flex-col gap-2">
                {analysis.tooShallow.map((grant) => (
                  <li
                    key={grant.permission}
                    className="flex flex-col gap-2 rounded-lg border border-grayA-4 px-3 py-2.5"
                  >
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                      <div className="overflow-x-auto">
                        <UrnText value={grant.permission} />
                      </div>
                      <span className="ml-auto text-[11px] text-gray-10">
                        {sourceText(grant.sources)}
                      </span>
                    </div>
                    <p className="text-xs text-gray-10">{grant.hint}</p>
                    <div className="flex flex-wrap items-center gap-3 rounded-md bg-grayA-2 px-2.5 py-2">
                      <span className="text-[11px] uppercase tracking-wide text-gray-9">
                        Suggested
                      </span>
                      <div className="overflow-x-auto">
                        <UrnText value={grant.suggestedPermission} />
                      </div>
                      <Button
                        variant="outline"
                        size="sm"
                        className="ml-auto"
                        onClick={() => onRequestGrant(grant.suggestedPermission)}
                      >
                        Draft fix
                      </Button>
                    </div>
                  </li>
                ))}
              </ul>
            </section>
          )}

          {analysis.unrelated.length > 0 && (
            <CollapsedGrantList
              summary={
                analysis.unrelated.length === 1
                  ? "1 unrelated grant"
                  : `${analysis.unrelated.length} unrelated grants`
              }
              grants={analysis.unrelated}
            />
          )}
        </div>
      )}
    </div>
  );
}

function CollapsedGrantList({ summary, grants }: { summary: string; grants: AnalyzedGrant[] }) {
  return (
    <details className="group rounded-lg border border-grayA-4">
      <summary className="cursor-pointer select-none px-3 py-2 text-xs text-gray-10 transition-colors hover:bg-grayA-2 group-open:border-b group-open:border-grayA-3">
        {summary}
      </summary>
      <ul className="flex flex-col divide-y divide-grayA-3">
        {grants.map((grant) => (
          <li
            key={grant.permission}
            className="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2 opacity-60"
          >
            <div className="overflow-x-auto">
              <UrnText value={grant.permission} />
            </div>
            <span className="ml-auto text-[11px] text-gray-10">{sourceText(grant.sources)}</span>
          </li>
        ))}
      </ul>
    </details>
  );
}
