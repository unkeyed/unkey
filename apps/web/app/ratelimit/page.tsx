"use client";
import * as React from "react";

import { LineChart } from "@/components/dashboard/charts";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Loading } from "@/components/dashboard/loading";
import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import Link from "next/link";
import { parseAsInteger, parseAsStringEnum, useQueryState } from "nuqs";
import { useLocalStorage } from "usehooks-ts";
export const dynamic = "force-dynamic";
export const runtime = "edge";

export type Data = {
  time: number;
  upstash: {
    latency: number;
    success: boolean;
    limit: number;
    remaining: number;
    reset: number;
  };
  unkeySync: {
    latency: number;
    success: boolean;
    limit: number;
    remaining: number;
    reset: number;
  };
  unkeyAsync: {
    latency: number;
    success: boolean;
    limit: number;
    remaining: number;
    reset: number;
  };
};

export default function RatelimitPage() {
  const [isTesting, setTesting] = React.useState(false);
  const [isResetting, setResetting] = React.useState(false);
  const [data, setData] = useLocalStorage<Data[]>("unkey-ratelimit-demo-compare", []);
  const [_reset, setReset] = React.useState<number | null>(null);
  React.useEffect(() => {
    const last = data.at(-1);
    if (!last) {
      setReset(null);
      return;
    }
  }, [data]);

  const [limit, setLimit] = useQueryState(
    "limit",
    parseAsInteger.withDefault(10).withOptions({
      history: "push",
    }),
  );

  const [duration, setDuration] = useQueryState(
    "duration",
    parseAsStringEnum(["10s", "60s", "5m"]).withDefault("10s").withOptions({
      history: "push",
    }),
  );

  async function test(): Promise<void> {
    setTesting(true);
    await fetch("/ratelimit/test", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        limit,
        duration,
      }),
    })
      .then((r) => r.json())
      .then((r) => {
        if (r) {
          setData([...data, r]);
        }
      })
      .finally(() => setTesting(false));
  }
  async function resetLimit(): Promise<void> {
    setResetting(true);
    await fetch("/ratelimit/reset", {
      method: "POST",
    }).finally(() => setResetting(false));
    setData([]);
  }
  const chartData = React.useMemo(() => {
    return data.filter(Boolean).flatMap((d) => [
      {
        x: new Date(d.time).toISOString(),
        y: d.unkeyAsync.latency,
        category: "unkey-async",
      },
      {
        x: new Date(d.time).toISOString(),
        y: d.unkeySync.latency,
        category: "unkey-sync",
      },
      {
        x: new Date(d.time).toISOString(),
        y: d.upstash.latency,
        category: "redis",
      },
    ]);
  }, [data]);

  const Chart = React.memo(LineChart);
  return (
    <div className="container relative pb-16 mx-auto">
      <div className="sticky top-0 py-4 bg-background">
        <PageHeader
          title="Ratelimit demo"
          description="Measuring latency between the Vercel Edge function, that is closest to you, and the ratelimit service"
          actions={[
            <Link href="/app" key="app">
              <Button>Sign In</Button>
            </Link>,
          ]}
        />
      </div>

      <main className="flex flex-col gap-4 mt-8 mb-20">
        <div className="flex items-end justify-center gap-4">
          <div key="limit" className="flex flex-col gap-1">
            <Label>Limit</Label>
            <Input
              type="number"
              value={limit}
              onChange={(e) => setLimit(Number.parseInt(e.currentTarget.value))}
            />
          </div>
          <div key="duration" className="flex flex-col gap-1">
            <Label>Duration</Label>
            <Select
              value={duration}
              onValueChange={(d) => {
                setDuration(d as any);
              }}
            >
              <SelectTrigger>
                <SelectValue defaultValue={limit} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="10s">10s</SelectItem>
                <SelectItem value="60s">60s</SelectItem>
                <SelectItem value="5m">5m</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button type="button" key="test" onClick={() => test()} disabled={isTesting}>
            {isTesting ? <Loading /> : "Test"}
          </Button>
          <Button type="button" key="reset" onClick={() => resetLimit()} disabled={isResetting}>
            {isResetting ? <Loading /> : "Reset"}
          </Button>
        </div>
        <div>
          {data.length > 0 ? (
            <div className="flex flex-col-reverse gap-4 sm:flex-row">
              <div className="flex flex-col gap-4 sm:w-1/2">
                <Card>
                  <CardHeader>
                    <CardTitle>Unkey Async</CardTitle>
                  </CardHeader>
                  <CardContent className="grid grid-cols-4 divide-x">
                    <Metric
                      label="Result"
                      value={
                        data.at(-1)!.unkeyAsync.success ? (
                          "Pass"
                        ) : (
                          <span className="text-alert">Ratelimited</span>
                        )
                      }
                    />
                    <Metric label="Remaining" value={data.at(-1)!.unkeyAsync.remaining} />

                    <Metric label="Limit" value={data.at(-1)!.unkeyAsync.limit} />

                    <Metric
                      label="Latency"
                      value={`${Math.round(data.at(-1)!.unkeyAsync.latency ?? 0)} ms`}
                    />
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle>Unkey Sync</CardTitle>
                  </CardHeader>
                  <CardContent className="grid grid-cols-4 divide-x">
                    <Metric
                      label="Result"
                      value={
                        data.at(-1)!.unkeySync.success ? (
                          "Pass"
                        ) : (
                          <span className="text-alert">Ratelimited</span>
                        )
                      }
                    />
                    <Metric label="Remaining" value={data.at(-1)!.unkeySync.remaining} />

                    <Metric label="Limit" value={data.at(-1)!.unkeySync.limit} />

                    <Metric
                      label="Latency"
                      value={`${Math.round(data.at(-1)!.unkeySync.latency ?? 0)} ms`}
                    />
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader>
                    <CardTitle>Redis</CardTitle>
                  </CardHeader>
                  <CardContent className="grid grid-cols-4 divide-x">
                    <Metric
                      label="Result"
                      value={
                        data.at(-1)!.upstash.success ? (
                          "Pass"
                        ) : (
                          <span className="text-alert">Ratelimited</span>
                        )
                      }
                    />
                    <Metric label="Remaining" value={data.at(-1)!.upstash.remaining} />

                    <Metric label="Limit" value={data.at(-1)!.upstash.limit} />

                    <Metric
                      label="Latency"
                      value={`${Math.round(data.at(-1)!.upstash.latency ?? 0)} ms`}
                    />
                  </CardContent>
                </Card>
              </div>
              <Card className="w-full sm:w-1/2">
                <CardHeader>
                  <CardTitle>Ratelimit latency (lower is better)</CardTitle>
                </CardHeader>
                <CardContent>
                  <Chart data={chartData} />
                </CardContent>
              </Card>
            </div>
          ) : (
            <EmptyPlaceholder>
              <EmptyPlaceholder.Title>No data</EmptyPlaceholder.Title>
              <EmptyPlaceholder.Description>Run a test first</EmptyPlaceholder.Description>
            </EmptyPlaceholder>
          )}
        </div>
      </main>
    </div>
  );
}

const Metric: React.FC<{
  label: React.ReactNode;
  value: React.ReactNode;
}> = ({ label, value }) => {
  return (
    <div className="flex flex-col items-start justify-center px-4 py-2">
      <p className="text-sm text-content-subtle">{label}</p>
      <div className="text-2xl font-semibold leading-none tracking-tight">{value}</div>
    </div>
  );
};
