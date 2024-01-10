import { BarChart, Callout, Card, Divider, Metric, Text } from "@tremor/react";
import { Unkey } from "@unkey/api";
import { FilterDateRange } from "./filter";
const unkey = new Unkey({ rootKey: process.env.UNKEY_ROOT_KEY! });
import { auth } from "@/auth";

type Props = {
  searchParams: {
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

  const t = new Date();
  const start = props.searchParams.start ? parseInt(props.searchParams.start) : t.getTime();
  t.setUTCMonth(t.getUTCMonth() - 1);
  const end = props.searchParams.end ? parseInt(props.searchParams.end) : Date.now();

  const verifications = (await unkey.keys.getVerifications({ ownerId, start, end })) as {
    error?: { message: string };
    result: {
      verifications: Array<{
        time: string;
        success: number;
        rateLimited: number;
        usageExceeded: number;
      }>;
    };
  };

  if (verifications.error) {
    throw new Error(`Error loading verifications: ${verifications.error.message}`);
  }

  const verificationsFormatted = verifications.result.verifications.map((v) => ({
    time: new Date(v.time).toDateString(),
    Success: v.success,
    "Rate Limited": v.rateLimited,
    "Usage Exceeded": v.usageExceeded,
  }));

  return (
    <>
      <FilterDateRange />
      <Divider />
      <Card>
        <Text>Title</Text>
        <Metric>Usage</Metric>
        <BarChart
          stack
          className="mt-6"
          data={verificationsFormatted}
          categories={["Success", "Rate Limited", "Usage Exceeded"]}
          index="time"
          yAxisWidth={48}
        />
      </Card>
    </>
  );
}
