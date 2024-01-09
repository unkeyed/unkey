import { BarChart, BarList, Callout, Card, Col, Divider, Grid, Metric, Text } from "@tremor/react";
import { Unkey } from "@unkey/api";
import { FilterDateRange } from "./filter";
const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY! });
import { auth } from "@/auth";

type Props = {
  searchParams: {
    keyId?: string;
    ownerId?: string;
    start?: string;
    end?: string;
  };
};

export default async function AnalyticsPage(props: Props) {
  const sess = await auth();
  const ownerId = sess?.user?.id;

  const allKeys = await unkey.apis.listKeys({
    apiId: process.env.UNKEY_API_ID!,
    ownerId,
  });
  if (allKeys.error) {
    return (
      <Callout title={allKeys.error.message} color="red">
        <pre>{JSON.stringify(allKeys.error, null, 2)}</pre>
      </Callout>
    );
  }
  if (props.searchParams.keyId) {
    allKeys.result.keys = allKeys.result.keys.filter((k) => k.id === props.searchParams.keyId);
  }
  if (props.searchParams.ownerId) {
    allKeys.result.keys = allKeys.result.keys.filter(
      (k) => k.ownerId && k.ownerId === props.searchParams.ownerId,
    );
  }

  const _keysByOwner = allKeys.result.keys.reduce((acc, key) => {
    if (!key.ownerId) {
      return acc;
    }
    if (!acc[key.ownerId]) {
      acc[key.ownerId] = 0;
    }
    acc[key.ownerId] += 1;
    return acc;
  }, {} as Record<string, number>);

  const remainingByOwner = allKeys.result.keys.reduce((acc, key) => {
    if (!key.id) {
      return acc;
    }
    if (!acc[key.id]) {
      acc[key.id] = 0;
    }
    acc[key.id] += key.remaining || 0;
    return acc;
  }, {} as Record<string, number>);

  const t = new Date();
  const start = props.searchParams.start ? parseInt(props.searchParams.start) : t.getTime();
  t.setUTCMonth(t.getUTCMonth() - 1);
  const end = props.searchParams.end ? parseInt(props.searchParams.end) : Date.now();

  // TODO: create a new API for this. parallel loading is not going to scale
  const keys = await Promise.all(
    allKeys.result.keys.map(async (key) => ({
      verifications: await getVerifications(key.id, start, end),
      // verifications: await unkey.keys.getVerifications({ keyId: key.id, start, end }),
      keyId: key.id,
    })),
  );

  const times = {} as Record<number, { [keyId: string]: number }>;
  for (const key of keys) {
    for (const verification of key.verifications) {
      if (!times[verification.time]) {
        times[verification.time] = {};
      }
      if (!times[verification.time][key.keyId]) {
        times[verification.time][key.keyId] = 0;
      }

      times[verification.time][key.keyId] += verification.success;
    }
  }
  const _verificationsAcrossAllkeys = keys.map((vs, i) => ({
    time: vs.verifications.at(i)?.time,
    success: vs.verifications.reduce((acc, v) => acc + v.success, 0),
  }));

  const data = Object.entries(times).map(([time, keys]) => ({
    time: new Date(parseInt(time)).toDateString(),
    ...keys,
  }));

  const categories = new Set<string>();
  Object.values(times).forEach((keys) => Object.keys(keys).forEach((key) => categories.add(key)));

  return (
    <>
      <Grid numItems={3} className="gap-2">
        <Col numColSpan={1}>
          <FilterDateRange />
        </Col>
      </Grid>
      <Divider />
      <Grid numItems={1} numItemsSm={2} numItemsLg={3} className="gap-2">
        <Col numColSpan={1} numColSpanLg={2}>
          <Card>
            <Text>Title</Text>
            <Metric>Usage</Metric>
            {keys.length > 0 ? (
              <BarChart
                stack
                className="mt-6"
                data={data}
                index="time"
                categories={keys.map((key) => key.keyId)}
                yAxisWidth={48}
              />
            ) : null}
          </Card>
        </Col>
        <Card>
          <Text>Total Keys</Text>
          <Metric>{allKeys.result.keys.length}</Metric>

          <BarList
            className="mt-2"
            data={Object.entries(remainingByOwner).map(([name, value]) => ({
              name,
              value,
            }))}
          />
        </Card>
        <Col />
      </Grid>
    </>
  );
}

async function getVerifications(keyId: string, start: number, end: number) {
  const url = new URL("https://api.unkey.dev/vx/keys.getVerifications");
  url.searchParams.set("keyId", keyId);
  url.searchParams.set("start", start.toString());
  url.searchParams.set("end", end.toString());
  const res = await fetch(url, {
    headers: {
      Authorization: `Bearer ${process.env.UNKEY_ROOT_KEY}`,
      "Content-Type": "application/json",
    },
  });
  const data = (await res.json()) as {
    verifications: {
      time: number;
      success: number;
      ratelimited: number;
      usageExceeded: number;
    }[];
  };

  console.log(data);

  return data.verifications;
}
