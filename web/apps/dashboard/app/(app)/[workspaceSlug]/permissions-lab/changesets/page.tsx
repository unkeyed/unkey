"use client";

/**
 * Concept: changesets. Permission edits staged as human-readable diffs and
 * applied or reverted like deployments. The composer builds a draft locally,
 * the timeline drives the shared lab store's stage/apply/revert lifecycle.
 */

import {
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import { useEffect, useRef } from "react";
import { perm } from "../lib/mock-data";
import { type ChangeOp, usePermissionsLab } from "../lib/store";
import { Composer } from "./composer";
import { Timeline } from "./timeline";

const SEED_TITLE = "Rotate payments access";
const SEED_OPS: ChangeOp[] = [
  {
    op: "add",
    principalID: "unkey_root_payments",
    permission: perm("keyspaces/ks_payments_prod/keys/*", "update_key"),
  },
  {
    op: "remove",
    principalID: "unkey_root_payments",
    permission: perm("keyspaces/ks_payments_prod/keys/*", "read_key"),
  },
  {
    op: "add_role",
    principalID: "unkey_root_ci",
    roleID: "role_key_minter",
  },
];

export default function ChangesetsPage() {
  const lab = usePermissionsLab();
  const labRef = useRef(lab);
  labRef.current = lab;
  const seeded = useRef(false);

  // Seed one staged (not applied) example so the timeline never starts empty.
  // Deferred a tick because the provider hydrates from localStorage in a mount
  // effect that runs after this one; deciding then avoids seeding over (or
  // being clobbered by) persisted state.
  useEffect(() => {
    const timer = window.setTimeout(() => {
      if (!seeded.current && labRef.current.state.changesets.length === 0) {
        seeded.current = true;
        labRef.current.stage(SEED_TITLE, SEED_OPS);
      }
    }, 0);
    return () => window.clearTimeout(timer);
  }, []);

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Changesets</PageHeaderTitle>
          <PageHeaderDescription>
            Stage permission edits as human-readable diffs, review their impact, then apply or
            revert them like deployments.
          </PageHeaderDescription>
        </PageHeaderContent>
      </PageHeader>
      <PageBody>
        <div className="grid grid-cols-1 items-start gap-8 p-6 xl:grid-cols-2">
          <section className="flex flex-col gap-4">
            <div className="flex flex-col gap-0.5">
              <h2 className="text-sm font-medium text-gray-12">Compose</h2>
              <p className="text-xs text-gray-11">
                Add changes one by one; nothing takes effect until the changeset is applied.
              </p>
            </div>
            <Composer />
          </section>
          <section className="flex flex-col gap-4">
            <div className="flex flex-col gap-0.5">
              <h2 className="text-sm font-medium text-gray-12">Timeline</h2>
              <p className="text-xs text-gray-11">
                Every changeset in this workspace, newest first. Reverted ones stay as history.
              </p>
            </div>
            <Timeline />
          </section>
        </div>
      </PageBody>
    </PageContainer>
  );
}
