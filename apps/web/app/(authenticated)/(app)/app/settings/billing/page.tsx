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
import { activeKeys, verifications } from "@/lib/tinybird";
import { cn } from "@/lib/utils";
import Link from "next/link";
import { redirect } from "next/navigation";
import { ChangePlan } from "./change-plan";

export const revalidate = 0;

export default async function BillingPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!workspace) {
    return redirect("/new");
  }
  const t = new Date();
  t.setUTCDate(1);
  t.setUTCHours(0, 0, 0, 0);

  const year = t.getUTCFullYear();
  const month = t.getUTCMonth() + 1;

  const [usedActiveKeys, usedVerifications] = await Promise.all([
    activeKeys({
      workspaceId: workspace.id,
      year,
      month,
    }).then((res) => res.data.at(0)?.keys ?? 0),
    verifications({
      workspaceId: workspace.id,
      year,
      month,
    }).then((res) => res.data.at(0)?.success ?? 0),
  ]);

  const activeKeysPercentage = percentage(usedActiveKeys, workspace.maxActiveKeys ?? 0);
  const verificationsPercentage = percentage(usedVerifications, workspace.maxVerifications ?? 0);

  return (
    <div className="flex flex-col gap-8 mb-20 ">
      <Card>
        <CardHeader className="flex flex-row items-start justify-between">
          <div>
            <CardTitle>Usage</CardTitle>
            <CardDescription>
              Current billing cycle:{" "}
              <span className="text-primary font-medium">
                {t.toLocaleString("en-US", { month: "long", year: "numeric" })}
              </span>{" "}
            </CardDescription>
          </div>
        </CardHeader>

        <CardContent>
          <Link href="/app/stripe">
            <Button className="max-sm:mb-4 max-sm:text-sm md:hidden">Change Billing</Button>
          </Link>
          <ol className="flex flex-col space-y-4">
            <li className="flex w-2/3 flex-col">
              <h3 className="text-content text-sm font-medium">Active Keys</h3>
              {activeKeysPercentage !== null ? (
                <div className="mt-1 overflow-hidden rounded-full bg-gray-300">
                  <div
                    className={cn("bg-primary h-2 rounded", {
                      "bg-alert": workspace.maxActiveKeys && activeKeysPercentage >= 100,
                    })}
                    style={{ width: `${activeKeysPercentage}%` }}
                  />
                </div>
              ) : null}
              <p className="text-content-subtle text-xs">
                {usedActiveKeys.toLocaleString()} /{" "}
                {workspace.maxActiveKeys?.toLocaleString() ?? "∞"}{" "}
                {activeKeysPercentage !== null
                  ? `(${activeKeysPercentage.toLocaleString(undefined, {
                      maximumFractionDigits: 2,
                    })}%)`
                  : null}
              </p>
            </li>
            <li className="flex w-2/3 flex-col">
              <h3 className="text-content text-sm font-medium">Verifications</h3>
              {verificationsPercentage !== null ? (
                <div className="mt-1 overflow-hidden rounded-full bg-gray-300">
                  <div
                    className={cn("bg-primary h-2 rounded", {
                      "bg-alert": workspace.maxVerifications && verificationsPercentage >= 100,
                    })}
                    style={{ width: `${verificationsPercentage}%` }}
                  />
                </div>
              ) : null}
              <p className="text-content-subtle text-xs">
                {usedVerifications} / {workspace.maxVerifications?.toLocaleString() ?? "∞"}{" "}
                {verificationsPercentage !== null
                  ? `(${verificationsPercentage.toLocaleString(undefined, {
                      maximumFractionDigits: 2,
                    })}%)`
                  : null}
              </p>
            </li>
          </ol>
        </CardContent>
        <CardFooter className="flex items-center justify-between">
          <p className="text-content-subtle text-xs">
            These are soft limits. We will not throttle or block you if you go over them, however we
            will contact you if you exceed them repeatedly.
          </p>
          <Link href="/app/settings/billing/stripe">
            <Button>Manage billing and invoices</Button>
          </Link>
        </CardFooter>
      </Card>

      <ChangePlan workspace={workspace} />
    </div>
  );
}

function percentage(num: number, total: number): number {
  if (total === 0) {
    return 0;
  }
  return Math.min(100, (num / total) * 100);
}
