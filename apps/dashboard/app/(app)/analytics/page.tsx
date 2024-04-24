import { AreaChart, StackedColumnChart } from "@/components/dashboard/charts";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { and, db, eq, isNull, schema, sql } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import {
  getAnalyticsActiveAll,
  getAnalyticsActiveDaily,
  getAnalyticsActiveHourly,
  getAnalyticsVerificationsDaily,
  getAnalyticsVerificationsHourly,
} from "@/lib/tinybird";
import { ChevronRight, Scan, User, VenetianMask } from "lucide-react";
import { redirect } from "next/navigation";
import {
  type ApiId,
  ApiIdSelect,
  type Interval,
  IntervalSelect,
  type OwnerId,
  OwnerIdSelect,
} from "./select";

import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import Link from "next/link";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function AnalyticsPage(props: {
  searchParams: {
    apiId?: ApiId;
    interval?: Interval;
    ownerId?: OwnerId;
  };
}) {
  const tenantId = getTenantId();

  // Setup the select options
  const apiIdSelect: { id: string; name: string }[] = [];
  const ownerIdSelect: { id: string; name: string }[] = [];
  const filteredKeysList: {
    id: string;
    name: string | null;
    keyAuthId: string;
    apiId: string;
    ownerId: string;
    start: string | null;
    roles: number;
    permissions: number;
    environment: string | null;
  }[] = [];
  ownerIdSelect.push({ id: "All", name: "All" });
  ownerIdSelect.push({ id: "Without_OwnerID", name: "Without OwnerID" });
  apiIdSelect.push({ id: "All", name: "All" });
  // Set search Params
  const interval = props.searchParams.interval ?? "7d";
  const ownerId = props.searchParams?.ownerId ?? "All";
  const apiId = props.searchParams?.apiId ?? "All";

  const t = new Date();
  t.setUTCDate(1);
  t.setUTCHours(0, 0, 0, 0);
  const billingCycleStart = t.getTime();
  const billingCycleEnd = t.setUTCMonth(t.getUTCMonth() + 1) - 1;
  // Get workspace and check if matches tenantId
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace || workspace.tenantId !== tenantId) {
    return redirect("/new");
  }

  //Get all Api's in this workspace and map to apiIdSelect and then to apiList
  const getApis = await db.query.apis.findMany({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.workspaceId, workspace.id), isNull(table.deletedAt)),
  });
  getApis.map((api) => {
    apiIdSelect.push({ id: api.id, name: api.name });
  });

  // Get all keys in this workspace and map to keysList and then to ownerIdSelect
  const getKeys = await db.query.keys.findMany({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.workspaceId, workspace.id), isNull(table.deletedAt)),
    limit: 100,
    with: {
      roles: true,
      permissions: true,
      keyAuth: {
        with: {
          api: true,
        },
      },
    },
  });
  const nullOwnerId = "Without_OwnerID";
  if (getKeys.length > 0) {
    const keyList = getKeys.map((key) => ({
      id: key.id,
      name: key.name,
      keyAuthId: key.keyAuth?.id,
      apiId: key.keyAuth?.api?.id,
      ownerId: key.ownerId ? key.ownerId : nullOwnerId,
      start: key.start,
      roles: key.roles.length,
      permissions: key.permissions.length,
      environment: key.environment,
    }));

    keyList.forEach((key) => {
      if (key.ownerId !== null) {
        if (!ownerIdSelect.some((e) => e.id === key.ownerId)) {
          ownerIdSelect.push({ id: key.ownerId, name: key.ownerId });
        }
      }
      if (ownerId === "All" || ownerId === key.ownerId) {
        if (apiId === "All" || apiId === key.apiId) {
          filteredKeysList.push({
            id: key.id,
            name: key.name ? key.name : null,
            keyAuthId: key.keyAuthId,
            apiId: key.apiId,
            ownerId: key.ownerId ? key.ownerId : nullOwnerId,
            start: key.start,
            roles: key.roles,
            permissions: key.permissions,
            environment: key.environment,
          });
        }
      }
    });
  }

  const { getVerificationsPerInterval, getActiveKeysPerInterval, start, end, granularity } =
    prepareInterval(interval);
  const [
    keys,
    verifications,
    activeKeys,
    activeKeysTotal,
    _activeKeysInBillingCycle,
    verificationsInBillingCycle,
  ] = await Promise.all([
    db
      .select({ count: sql<number>`count(*)` })
      .from(schema.keys)
      .where(and(eq(schema.keys.workspaceId, workspace.id!), isNull(schema.keys.deletedAt)))
      .execute()
      .then((res) => res.at(0)?.count ?? 0),
    getVerificationsPerInterval({
      workspaceId: workspace.id,
      ownerId: ownerId !== "All" ? ownerId : undefined,
      apiId: apiId !== "All" ? apiId : undefined,
      start,
      end,
    }),
    getActiveKeysPerInterval({
      workspaceId: workspace.id,
      ownerId: ownerId !== "All" ? ownerId : undefined,
      apiId: apiId !== "All" ? apiId : undefined,
      start,
      end,
    }),
    getAnalyticsActiveAll({
      workspaceId: workspace.id,
      apiId: apiId !== "All" ? apiId : undefined,
      ownerId: ownerId !== "All" ? ownerId : undefined,
      start: start,
      end: end,
    }),
    getActiveKeysPerInterval({
      workspaceId: workspace.id,
      ownerId: ownerId !== "All" ? ownerId : undefined,
      apiId: apiId !== "All" ? apiId : undefined,
      start: billingCycleStart,
      end: billingCycleEnd,
    }).then((res) => res.data.at(0)),
    getVerificationsPerInterval({
      workspaceId: workspace.id,
      ownerId: ownerId !== "All" ? ownerId : undefined,
      apiId: apiId !== "All" ? apiId : undefined,
      start: billingCycleStart,
      end: billingCycleEnd,
    }),
  ]);

  const successOverTime: { x: string; y: number }[] = [];
  const ratelimitedOverTime: { x: string; y: number }[] = [];
  const usageExceededOverTime: { x: string; y: number }[] = [];

  for (const d of verifications.data.sort((a, b) => a.time - b.time)) {
    const x = new Date(d.time).toISOString();
    successOverTime.push({ x, y: d.success });
    ratelimitedOverTime.push({ x, y: d.rateLimited });
    usageExceededOverTime.push({ x, y: d.usageExceeded });
  }

  const verificationsData = [
    ...successOverTime.map((d) => ({
      ...d,
      category: "Successful Verifications",
    })),
    ...ratelimitedOverTime.map((d) => ({ ...d, category: "Ratelimited" })),
    ...usageExceededOverTime.map((d) => ({ ...d, category: "Usage Exceeded" })),
  ];

  type KeysByOwnerId = {
    [ownerId: string]: {
      id: string;
      name: string | null;
      keyAuthId: string;
      apiId: string;
      ownerId: string | null;
      start: string | null;
      roles: number;
      permissions: number;
      environment: string | null;
    }[];
  };
  const keysByOwnerId = filteredKeysList.reduce((acc, curr) => {
    const ownerId = curr.ownerId ?? nullOwnerId;
    if (!acc[ownerId]) {
      acc[ownerId] = [];
    }
    acc[ownerId].push({
      id: curr.id,
      name: curr.name,
      keyAuthId: curr.keyAuthId,
      apiId: curr.apiId,
      ownerId: curr.ownerId,
      start: curr.start,
      roles: curr.roles,
      permissions: curr.permissions,
      environment: curr.environment,
    });
    return acc;
  }, {} as KeysByOwnerId);

  const activeKeysOverTime = activeKeys.data.map(({ time, keys }) => ({
    x: new Date(time).toISOString(),
    y: keys,
  }));

  return (
    <div className="flex flex-col gap-8 mb-20 w-full">
      <div className="flex flex-wrap gap-4 w-full">
        <div className="w-44">
          <p>Interval Select</p>
          <IntervalSelect defaultTimeSelected={interval} />
        </div>
        <div className="w-44">
          <p>OwnerId Select</p>
          <OwnerIdSelect defaultOwnerIdSelected={"All"} ownerIdList={ownerIdSelect} />
        </div>
        <div className="w-44">
          <p>Api Select</p>
          <ApiIdSelect defaultApiIdSelected={"All"} apiIdList={apiIdSelect} />
        </div>
      </div>
      <Separator className="my-2" />
      <Card>
        <CardContent className="grid grid-cols-3 divide-x max-sm:p-0">
          <Metric label="Total Keys" value={formatNumber(keys)} />
          <Metric
            label={`Verifications in ${new Date().toLocaleString("en-US", {
              month: "long",
            })}`}
            value={formatNumber(
              verificationsInBillingCycle.data.reduce((sum, day) => sum + day.success, 0),
            )}
          />
          <Metric
            label={`Active Keys in ${new Date().toLocaleString("en-US", {
              month: "long",
            })}`}
            value={formatNumber(activeKeysTotal.data.at(0)?.keys ?? 0)}
          />
        </CardContent>
      </Card>
      <div className="flex flex-col w-full gap-6">
        <div className="flex w-full">
          <Card className="w-full h-full">
            <div className="grid grid-cols-3 divide-x pl-8">
              <Metric
                label="Successful Verifications"
                value={formatNumber(verifications.data.reduce((sum, day) => sum + day.success, 0))}
              />
              <Metric
                label="Ratelimited"
                value={formatNumber(
                  verifications.data.reduce((sum, day) => sum + day.rateLimited, 0),
                )}
              />
              <Metric
                label="Usage Exceeded"
                value={formatNumber(
                  verifications.data.reduce((sum, day) => sum + day.usageExceeded, 0),
                )}
              />
            </div>
            <CardContent>
              <StackedColumnChart
                colors={["primary", "warn", "danger"]}
                data={verificationsData}
                timeGranularity={
                  granularity >= 1000 * 60 * 60 * 24 * 30
                    ? "month"
                    : granularity >= 1000 * 60 * 60 * 24
                      ? "day"
                      : "hour"
                }
              />
            </CardContent>
          </Card>
        </div>
        <div className="flex w-full ">
          <Card className="w-full h-full">
            <CardContent>
              <AreaChart
                data={activeKeysOverTime}
                tooltipLabel="Active Keys"
                timeGranularity={
                  granularity >= 1000 * 60 * 60 * 24 * 30
                    ? "month"
                    : granularity >= 1000 * 60 * 60 * 24
                      ? "day"
                      : "hour"
                }
              />
            </CardContent>
          </Card>
        </div>
      </div>
      <div className="flex">
        {filteredKeysList.length === 0 ? (
          <EmptyPlaceholder>
            <EmptyPlaceholder.Icon>
              <Scan />
            </EmptyPlaceholder.Icon>
            <EmptyPlaceholder.Title>No keys found</EmptyPlaceholder.Title>
            <EmptyPlaceholder.Description>Create your first key</EmptyPlaceholder.Description>
            {/* <CreateNewRole trigger={<Button variant="primary">Create New Role</Button>} /> */}
          </EmptyPlaceholder>
        ) : (
          Object.entries(keysByOwnerId).map(([ownerId, ks]) => (
            <div key={ownerId} className="flex flex-col gap-2 w-full">
              <div className="items-left gap-1 flex-row">
                <Collapsible>
                  <CollapsibleTrigger className="border rounded-lg w-full ">
                    {ownerId === nullOwnerId ? (
                      <div className="flex flex-row gap-8 items-center px-4 py-2 duration-250 hover:bg-background-subtle ">
                        <VenetianMask className="w-4 h-4 text-content" />
                        Without OwnerID
                      </div>
                    ) : (
                      <div
                        key="apiId"
                        className="grid items-center grid-cols-12 px-4 py-2 duration-250 hover:bg-background-subtle "
                      >
                        <User className="w-4 h-4 text-content items-start" />
                        {ownerId}
                      </div>
                    )}
                  </CollapsibleTrigger>
                  <CollapsibleContent>
                    <ul className="flex flex-col overflow-hidden border divide-y rounded-lg divide-border bg-background border-borde w-fullr">
                      {ks.map((k) => (
                        <Link
                          href={`/app/keys/${k.keyAuthId}/${k.id}`}
                          key={k.id}
                          className="grid items-center grid-cols-12 px-4 py-2 duration-250 hover:bg-background-subtle w-full"
                        >
                          <div className="flex flex-col items-start col-span-10 md:col-span-6 ">
                            <span className="text-sm text-content">{k.name}</span>
                            <pre className="text-xs text-content-subtle">{k.id}</pre>
                          </div>

                          <div className="flex items-center col-span-12 md:col-span-2 gap-2 max-sm:truncate">
                            {k.environment ? (
                              <Badge key={k.environment} variant="secondary">
                                env: {k.environment}
                              </Badge>
                            ) : null}
                          </div>

                          <div className="flex items-center col-span-10 md:col-span-2 gap-2 w-full">
                            <Badge variant="secondary">
                              {Intl.NumberFormat(undefined, {
                                notation: "compact",
                              }).format(k.permissions)}{" "}
                              Permission
                              {k.permissions !== 1 ? "s" : ""}
                            </Badge>

                            <Badge variant="secondary">
                              {Intl.NumberFormat(undefined, {
                                notation: "compact",
                              }).format(k.roles)}{" "}
                              Role
                              {k.roles !== 1 ? "s" : ""}
                            </Badge>
                          </div>

                          <div className="flex items-center justify-end col-span-2 md:row-span-2">
                            <Button variant="ghost">
                              <ChevronRight className="w-4 h-4" />
                            </Button>
                          </div>
                        </Link>
                      ))}
                    </ul>
                  </CollapsibleContent>
                </Collapsible>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function prepareInterval(interval: Interval) {
  const now = new Date();

  switch (interval) {
    case "24h": {
      const end = now.setUTCHours(now.getUTCHours() + 1, 0, 0, 0);
      const intervalMs = 1000 * 60 * 60 * 24;
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60 * 60,
        getVerificationsPerInterval: getAnalyticsVerificationsHourly,
        getActiveKeysPerInterval: getAnalyticsActiveHourly,
      };
    }
    case "7d": {
      now.setUTCDate(now.getUTCDate() + 1);
      const end = now.setUTCHours(0, 0, 0, 0);
      const intervalMs = 1000 * 60 * 60 * 24 * 7;
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60 * 60 * 24,
        getVerificationsPerInterval: getAnalyticsVerificationsDaily,
        getActiveKeysPerInterval: getAnalyticsActiveDaily,
      };
    }
    case "30d": {
      now.setUTCDate(now.getUTCDate() + 1);
      const end = now.setUTCHours(0, 0, 0, 0);
      const intervalMs = 1000 * 60 * 60 * 24 * 30;
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60 * 60 * 24,
        getVerificationsPerInterval: getAnalyticsVerificationsDaily,
        getActiveKeysPerInterval: getAnalyticsActiveDaily,
      };
    }
    case "90d": {
      now.setUTCDate(now.getUTCDate() + 1);
      const end = now.setUTCHours(0, 0, 0, 0);
      const intervalMs = 1000 * 60 * 60 * 24 * 90;
      return {
        start: end - intervalMs,
        end,
        intervalMs,
        granularity: 1000 * 60 * 60 * 24,
        getVerificationsPerInterval: getAnalyticsVerificationsDaily,
        getActiveKeysPerInterval: getAnalyticsActiveDaily,
      };
    }
  }
}

const Metric: React.FC<{ label: string; value: string }> = ({ label, value }) => {
  return (
    <div className="flex flex-col items-start justify-center px-2 md:px-4 py-1 md:py-2 overflow-hidden">
      <p className="flex text-xs md:text-sm text-content-subtle truncate">{label}</p>
      <div className="text-md md:text-2xl h-1/2 font-semibold leading-none tracking-tight">
        {value}
      </div>
    </div>
  );
};
