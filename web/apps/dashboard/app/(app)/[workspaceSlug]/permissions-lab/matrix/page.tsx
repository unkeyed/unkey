"use client";

import { Check, Magnifier, Plus } from "@unkey/icons";
import {
  ConfirmPopover,
  Empty,
  Input,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import { useMemo, useRef, useState } from "react";
import { type ConcreteResource, perm } from "../lib/mock-data";
import { type GrantOp, usePermissionsLab } from "../lib/store";
import { FamilySection } from "./family-section";
import {
  FAMILIES,
  type Family,
  type ParsedGrant,
  type PatternRow,
  buildGrantIndex,
  patternRowsForFamily,
} from "./matrix-model";

/** Sections with more rows than this start collapsed. */
const COLLAPSE_THRESHOLD = 10;

interface PendingConfirm {
  title: string;
  description: string;
  variant: "warning" | "danger";
  confirmButtonText: string;
  commitTitle: string;
  ops: GrantOp[];
  successMessage: string;
}

export default function MatrixPage() {
  const lab = usePermissionsLab();
  const principals = lab.state.principals;

  const [principalID, setPrincipalID] = useState(principals[0]?.id ?? "");
  const [query, setQuery] = useState("");
  const [expandedOverrides, setExpandedOverrides] = useState<Record<string, boolean>>({});

  const confirmAnchorRef = useRef<HTMLElement | null>(null);
  const [pending, setPending] = useState<PendingConfirm | null>(null);

  const principal = principals.find((p) => p.id === principalID) ?? principals[0];

  const rolesByID = useMemo(
    () => new Map(lab.state.roles.map((r) => [r.id, r])),
    [lab.state.roles],
  );

  // Parse every grant once per principal; cells resolve against this index.
  const grants = useMemo<ParsedGrant[]>(
    () => (principal ? buildGrantIndex(principal, rolesByID) : []),
    [principal, rolesByID],
  );

  const patternRowsByFamily = useMemo(
    () => new Map(FAMILIES.map((family) => [family.type, patternRowsForFamily(grants, family)])),
    [grants],
  );

  const searching = query.trim().length > 0;
  const filteredRowsByFamily = useMemo(() => {
    const q = query.trim().toLowerCase();
    return new Map(
      FAMILIES.map((family) => [
        family.type,
        q === ""
          ? family.resources
          : family.resources.filter(
              (r) => r.label.toLowerCase().includes(q) || r.path.toLowerCase().includes(q),
            ),
      ]),
    );
  }, [query]);

  const matchCount = useMemo(
    () => [...filteredRowsByFamily.values()].reduce((sum, rows) => sum + rows.length, 0),
    [filteredRowsByFamily],
  );

  const directCount = principal?.permissions.length ?? 0;
  const roleGrantCount = grants.filter((g) => g.sources.every((s) => s.kind !== "direct")).length;

  const openConfirm = (anchor: HTMLElement, confirm: PendingConfirm) => {
    confirmAnchorRef.current = anchor;
    setPending(confirm);
  };

  const handleGrant = (resource: ConcreteResource, action: string) => {
    if (!principal) {
      return;
    }
    const permission = perm(resource.path, action);
    lab.commit(`Grant ${action} on ${resource.label}`, [
      { op: "add", principalID: principal.id, permission },
    ]);
    toast.success(`Granted ${action} on ${resource.label}`);
  };

  const handleRequestRevoke = (
    anchor: HTMLElement,
    resource: ConcreteResource,
    action: string,
    grant: ParsedGrant,
  ) => {
    if (!principal) {
      return;
    }
    openConfirm(anchor, {
      title: "Revoke permission",
      description: `Removes ${resource.path}#${action} from ${principal.name}. Access inherited from patterns or roles is not affected.`,
      variant: "danger",
      confirmButtonText: "Revoke",
      commitTitle: `Revoke ${action} on ${resource.label}`,
      ops: [{ op: "remove", principalID: principal.id, permission: grant.raw }],
      successMessage: `Revoked ${action} on ${resource.label}`,
    });
  };

  const handleRequestMaterialize = (anchor: HTMLElement, row: PatternRow) => {
    if (!principal) {
      return;
    }
    const action = row.grant.permission.action;
    const family = FAMILIES.find((f) => patternRowsByFamily.get(f.type)?.includes(row));
    const familyLabel = family ? family.title.toLowerCase() : "resources";
    const adds = row.covered
      .map((r) => perm(r.path, action))
      .filter((p) => !principal.permissions.includes(p));

    openConfirm(anchor, {
      title: "Materialize pattern grant",
      description: `Removes 1 pattern grant and adds ${adds.length} concrete ${
        adds.length === 1 ? "grant" : "grants"
      } for the ${familyLabel} it currently covers. Resources created later will not be covered anymore.`,
      variant: "warning",
      confirmButtonText: "Materialize",
      commitTitle: `Materialize ${row.grant.permission.urn.resource}#${action}`,
      ops: [
        { op: "remove", principalID: principal.id, permission: row.grant.raw },
        ...adds.map(
          (permission): GrantOp => ({ op: "add", principalID: principal.id, permission }),
        ),
      ],
      successMessage: `Materialized into ${adds.length} concrete ${
        adds.length === 1 ? "grant" : "grants"
      }`,
    });
  };

  const handleConfirm = () => {
    if (!pending) {
      return;
    }
    lab.commit(pending.commitTitle, pending.ops);
    toast.success(pending.successMessage);
  };

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>The Matrix</PageHeaderTitle>
          <PageHeaderDescription>
            Every resource by every action, one family at a time. Click a cell to grant or revoke
            the exact permission; inherited access explains itself on hover.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        {principal ? (
          <>
            <div className="sticky top-0 z-20 flex flex-wrap items-center gap-3 border-b border-grayA-4 bg-base-12 px-6 py-3">
              <div className="w-[280px]">
                <Select
                  value={principal.id}
                  items={principals.map((p) => ({ value: p.id, label: p.name }))}
                  onValueChange={(value) => {
                    if (value !== null) {
                      setPrincipalID(value);
                    }
                  }}
                >
                  <SelectTrigger aria-label="Principal">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {principals.map((p) => (
                        <SelectItem key={p.id} value={p.id}>
                          <span className="flex items-baseline gap-2">
                            <span>{p.name}</span>
                            <span className="font-mono text-[11px] text-gray-9">{p.id}</span>
                          </span>
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>
              <div className="w-full max-w-sm">
                <Input
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="Filter resources by name, path, or id"
                  leftIcon={<Magnifier iconSize="sm-regular" />}
                  aria-label="Filter resources"
                />
              </div>
              <span className="text-xs text-gray-10">
                {searching
                  ? `${matchCount} ${matchCount === 1 ? "resource matches" : "resources match"}`
                  : `${directCount} direct ${directCount === 1 ? "grant" : "grants"}, ${roleGrantCount} via roles`}
              </span>
            </div>

            <div className="flex flex-col gap-4 p-6">
              <Legend />

              {searching && matchCount === 0 ? (
                <Empty>
                  <Empty.Title>No resources match</Empty.Title>
                  <Empty.Description>
                    Nothing in the ACME workspace matches &quot;{query.trim()}&quot;. Try a resource
                    name, a path segment, or an id.
                  </Empty.Description>
                </Empty>
              ) : (
                FAMILIES.map((family) => {
                  const rows = filteredRowsByFamily.get(family.type) ?? [];
                  const patternRows = patternRowsByFamily.get(family.type) ?? [];
                  if (searching && rows.length === 0) {
                    return null;
                  }
                  return (
                    <FamilySection
                      key={family.type}
                      family={family}
                      rows={rows}
                      patternRows={patternRows}
                      grants={grants}
                      searching={searching}
                      expanded={isExpanded(family, rows, searching, expandedOverrides)}
                      onToggle={() =>
                        setExpandedOverrides((prev) => ({
                          ...prev,
                          [family.type]: !isExpanded(family, rows, false, prev),
                        }))
                      }
                      onGrant={handleGrant}
                      onRequestRevoke={handleRequestRevoke}
                      onRequestMaterialize={handleRequestMaterialize}
                    />
                  );
                })
              )}
            </div>
          </>
        ) : (
          <div className="p-6">
            <Empty>
              <Empty.Title>No principals</Empty.Title>
              <Empty.Description>
                The mock workspace has no root keys to manage. Reset the lab data from the overview
                page to restore the seed.
              </Empty.Description>
            </Empty>
          </div>
        )}
      </PageBody>

      <ConfirmPopover
        isOpen={pending !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPending(null);
          }
        }}
        onConfirm={handleConfirm}
        triggerRef={confirmAnchorRef}
        title={pending?.title ?? "Confirm"}
        description={pending?.description ?? ""}
        confirmButtonText={pending?.confirmButtonText ?? "Confirm"}
        variant={pending?.variant ?? "warning"}
      />
    </PageContainer>
  );
}

function isExpanded(
  family: Family,
  rows: ConcreteResource[],
  searching: boolean,
  overrides: Record<string, boolean>,
): boolean {
  if (searching) {
    return true;
  }
  return overrides[family.type] ?? rows.length <= COLLAPSE_THRESHOLD;
}

function Legend() {
  return (
    <div className="flex flex-wrap items-center gap-x-6 gap-y-2 rounded-lg border border-grayA-4 bg-grayA-2 px-4 py-2.5 text-xs text-gray-11">
      <span className="flex items-center gap-2">
        <span className="inline-flex size-6 items-center justify-center rounded-md bg-successA-3 text-success-11">
          <Check iconSize="sm-regular" />
        </span>
        Granted directly. Click to revoke.
      </span>
      <span className="flex items-center gap-2">
        <span className="inline-flex size-6 items-center justify-center rounded-md border border-successA-5 text-success-9">
          <Check iconSize="sm-regular" />
        </span>
        Covered by a pattern or role. Hover for the source.
      </span>
      <span className="flex items-center gap-2">
        <span className="inline-flex size-6 items-center justify-center rounded-md border border-dashed border-grayA-5 text-gray-9">
          <Plus iconSize="sm-regular" />
        </span>
        Not granted. Click to grant this exact resource.
      </span>
      <span className="flex items-center gap-2">
        <span className="inline-flex h-6 w-9 items-center justify-center rounded-md bg-warningA-3 font-mono text-[11px] text-warning-11">
          *
        </span>
        Pattern grant. Spans the section, materializes into concrete grants.
      </span>
    </div>
  );
}
