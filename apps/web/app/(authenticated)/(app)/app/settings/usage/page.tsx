import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { cn } from "@/lib/utils";

import Link from "next/link";
import { redirect } from "next/navigation";

export const revalidate = 0;

export default async function SettingsPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!workspace) {
    return redirect("/onboarding");
  }

  const t = new Date();
  t.setUTCDate(0);
  t.setUTCHours(0, 0, 0, 0);

  let start = t.getTime() + 1;

  t.setUTCMonth(t.getUTCMonth() + 1);
  let end = t.getTime();

  if (workspace.billingPeriodStart) {
    start = workspace.billingPeriodStart.getTime();
  }
  if (workspace.billingPeriodEnd) {
    end = workspace.billingPeriodEnd.getTime();
  }

  const activeKeysPercentage = percentage(
    workspace.usageActiveKeys ?? 0,
    workspace.maxActiveKeys ?? 0,
  );
  const verificationsPercentage = percentage(
    workspace.usageVerifications ?? 0,
    workspace.maxVerifications ?? 0,
  );

  return (
    <div>
      <Card>
        <CardHeader className="flex flex-row items-start justify-between">
          <div>
            <CardTitle>Usage</CardTitle>
            <CardDescription>
              Current billing cycle:{" "}
              <span className="font-medium text-primary">
                {new Date(start).toDateString()} - {new Date(end).toDateString()}
              </span>{" "}
            </CardDescription>
          </div>
          <Link href="/app/stripe">
            <Button className="max-sm:text-xs">Change Billing</Button>
          </Link>
        </CardHeader>

        <CardContent>
          <ol className="flex flex-col space-y-4">
            <li className="flex flex-col w-2/3">
              <h3 className="text-sm font-medium text-content">Active Keys</h3>
              {activeKeysPercentage !== null ? (
                <div className="mt-1 overflow-hidden bg-gray-300 rounded-full">
                  <div
                    className={cn("h-2 rounded bg-primary", {
                      "bg-alert": workspace.maxActiveKeys && activeKeysPercentage >= 100,
                    })}
                    style={{ width: `${activeKeysPercentage}%` }}
                  />
                </div>
              ) : null}
              <p className="text-xs text-content-subtle">
                {workspace.usageActiveKeys?.toLocaleString()} /{" "}
                {workspace.maxActiveKeys?.toLocaleString() ?? "∞"}{" "}
                {activeKeysPercentage !== null
                  ? `(${activeKeysPercentage.toLocaleString(undefined, {
                      maximumFractionDigits: 2,
                    })}%)`
                  : null}
              </p>
            </li>
            <li className="flex flex-col w-2/3">
              <h3 className="text-sm font-medium text-content">Verifications</h3>
              {verificationsPercentage !== null ? (
                <div className="mt-1 overflow-hidden bg-gray-300 rounded-full">
                  <div
                    className={cn("h-2 rounded bg-primary", {
                      "bg-alert": workspace.maxVerifications && verificationsPercentage >= 100,
                    })}
                    style={{ width: `${verificationsPercentage}%` }}
                  />
                </div>
              ) : null}
              <p className="text-xs text-content-subtle">
                {workspace.usageVerifications?.toLocaleString()} /{" "}
                {workspace.maxVerifications?.toLocaleString() ?? "∞"}{" "}
                {verificationsPercentage !== null
                  ? `(${verificationsPercentage.toLocaleString(undefined, {
                      maximumFractionDigits: 2,
                    })}%)`
                  : null}
              </p>
            </li>
          </ol>
        </CardContent>
        <CardFooter>
          <p className="text-xs text-content-subtle">
            These are soft limits. We will not throttle or block you if you go over them, however we
            will contact you if you exceed them repeatedly.
          </p>
        </CardFooter>
      </Card>
    </div>
  );
}

function percentage(num: number, total: number): number {
  if (total === 0) {
    return 0;
  }
  return Math.min(100, (num / total) * 100);
}
