"use client";
import * as React from "react";

import { AreaChart } from "@/components/dashboard/charts";
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
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Minus } from "lucide-react";
import ms from "ms";
import Link from "next/link";
import { parseAsBoolean, parseAsInteger, parseAsStringEnum, useQueryState } from "nuqs";
import { useLocalStorage } from "usehooks-ts";
export const dynamic = "force-dynamic";
export const runtime = "edge";

export type Data = {
  latency: number;
  success: boolean;
  limit: number;
  remaining: number;
  reset: number;
  time: number;
};

export default function RatelimitPage() {
  const [isTesting, setTesting] = React.useState(false);
  const [isResetting, setResetting] = React.useState(false);
  const [data, setData] = useLocalStorage<Data[]>("unkey-ratelimit-demo", []);
  const [reset, setReset] = React.useState<number | null>(null);
  React.useEffect(() => {
    const last = data.at(-1);
    if (!last) {
      setReset(null);
      return;
    }

    const id = setInterval(() => {
      setReset(Math.max(0, last.reset - Date.now()));
    }, 100);

    return () => {
      clearInterval(id);
    };
  }, [data]);

  const [limit, setLimit] = useQueryState(
    "limit",
    parseAsInteger.withDefault(10).withOptions({
      history: "push",
    }),
  );
  const [async, setAsync] = useQueryState(
    "async",
    parseAsBoolean.withDefault(true).withOptions({
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
        async,
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
    return data.filter(Boolean).map((d) => ({
      x: new Date(d.time).toISOString(),
      y: d.latency,
    }));
  }, [data]);

  return (
    <div className="container relative pb-16 mx-auto">
      <div className="sticky top-0 py-4 bg-background">
        <PageHeader
          title="Ratelimit demo"
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
              onChange={(e) => setLimit(parseInt(e.currentTarget.value))}
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
                <SelectItem value="1s">1s</SelectItem>
                <SelectItem value="10s">10s</SelectItem>
                <SelectItem value="60s">60s</SelectItem>
                <SelectItem value="5m">5m</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div key="async" className="flex flex-col gap-1">
            <Label>Async</Label>
            <Select
              value={async ? "async" : "sync"}
              onValueChange={(d) => {
                setAsync(d === "async");
              }}
            >
              <SelectTrigger>
                <SelectValue defaultValue={async.toString()} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="sync">Sync</SelectItem>
                <SelectItem value="async">Async</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button type="button" key="test" onClick={() => test()}>
            {isTesting ? <Loading /> : "Test"}
          </Button>
          <Button type="button" key="reset" onClick={() => resetLimit()}>
            {isResetting ? <Loading /> : "Reset"}
          </Button>
        </div>
        <TooltipProvider>
          {data.length > 0 ? (
            <Card>
              <CardContent className="grid grid-cols-6 divide-x">
                <Metric
                  label="Result"
                  value={
                    data.at(-1)!.success ? "Pass" : <span className="text-alert">Ratelimited</span>
                  }
                />
                <Metric label="Remaining" value={data.at(-1)!.remaining} />

                <Metric label="Limit" value={data.at(-1)!.limit} />
                <Metric
                  label="Reset in"
                  value={reset ? ms(reset) : <Minus className="w-4 h-4" />}
                />
                <Metric
                  label="Latency"
                  value={`${Math.round(data.at(-1)!.latency ?? 0)} ms`}
                  tooltip="Latency is measured between the Next.js route and unkey, which is the real world scenario. The connection between your browser and the Next.js route may add additional overhead"
                />
              </CardContent>
            </Card>
          ) : null}

          <Card>
            <CardHeader>
              <CardTitle>Ratelimit latency in milliseconds</CardTitle>
            </CardHeader>
            <CardContent>
              {data.length > 0 ? (
                <AreaChart data={chartData} timeGranularity="hour" tooltipLabel="Latency [ms]" />
              ) : (
                <EmptyPlaceholder>
                  <EmptyPlaceholder.Title>No data</EmptyPlaceholder.Title>
                  <EmptyPlaceholder.Description>Run a test first</EmptyPlaceholder.Description>
                </EmptyPlaceholder>
              )}
            </CardContent>
          </Card>
        </TooltipProvider>
      </main>
    </div>
  );
}

const Metric: React.FC<{
  label: React.ReactNode;
  value: React.ReactNode;
  tooltip?: React.ReactNode;
}> = ({ label, value, tooltip }) => {
  const component = (
    <div className="flex flex-col items-start justify-center px-4 py-2">
      <p className="text-sm text-content-subtle">{label}</p>
      <div className="text-2xl font-semibold leading-none tracking-tight">{value}</div>
    </div>
  );

  if (tooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>{component}</TooltipTrigger>
        <TooltipContent>
          <p className="text-sm text-content-subtle">{tooltip}</p>
        </TooltipContent>
      </Tooltip>
    );
  }
  return component;
};
