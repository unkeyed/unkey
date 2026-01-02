import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { TimestampInfo } from "@unkey/ui";

export const TimestampExampleLocalTime = () => {
  const now = Date.now();
  const oneHourAgo = now - 3600 * 1000;
  const oneDayAgo = now - 86400 * 1000;
  const oneWeekAgo = now - 604800 * 1000;

  return (
    <div className="flex flex-col gap-4">
      <RenderComponentWithSnippet
        customCodeSnippet={`{/* Current Time */}
<div className="flex flex-col gap-2 text-center">
  <h3 className="text-sm font-medium">Current Time</h3>
  <TimestampInfo value={now} displayType="local" />
</div>
{/* One Hour Ago */}
<div className="flex flex-col gap-2 text-center">
  <h3 className="text-sm font-medium">One Hour Ago</h3>
  <TimestampInfo value={oneHourAgo} displayType="local" />
</div>
{/* One Day Ago */}
<div className="flex flex-col gap-2 text-center">
  <h3 className="text-sm font-medium">One Day Ago</h3>
  <TimestampInfo value={oneDayAgo} displayType="local" />
</div>
{/* One Week Ago */}
<div className="flex flex-col gap-2 text-center">
  <h3 className="text-sm font-medium">One Week Ago</h3>
  <TimestampInfo value={oneWeekAgo} displayType="local" />
</div>`}
      >
        <Row>
          {/* Current Time */}
          <div className="flex flex-col gap-2 text-center">
            <h3 className="text-sm font-medium">Current Time</h3>
            <TimestampInfo value={now} displayType="local" />
          </div>
          {/* One Hour Ago */}
          <div className="flex flex-col gap-2 text-center">
            <h3 className="text-sm font-medium">One Hour Ago</h3>
            <TimestampInfo value={oneHourAgo} displayType="local" />
          </div>
          {/* One Day Ago */}
          <div className="flex flex-col gap-2 text-center">
            <h3 className="text-sm font-medium">One Day Ago</h3>
            <TimestampInfo value={oneDayAgo} displayType="local" />
          </div>
          {/* One Week Ago */}
          <div className="flex flex-col gap-2 text-center">
            <h3 className="text-sm font-medium">One Week Ago</h3>
            <TimestampInfo value={oneWeekAgo} displayType="local" />
          </div>
        </Row>
      </RenderComponentWithSnippet>
    </div>
  );
};

export const TimestampExampleUTC = () => {
  const now = Date.now();
  const oneHourAgo = now - 3600 * 1000;
  const oneDayAgo = now - 86400 * 1000;
  const oneWeekAgo = now - 604800 * 1000;

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`{/* Current Time */}
<div className="flex flex-col gap-2 text-center">
  <h3 className="text-sm font-medium">Current Time</h3>
  <TimestampInfo value={now} displayType="utc" />
</div>
{/* One Hour Ago */}
<div className="flex flex-col gap-2 text-center">
  <h3 className="text-sm font-medium">One Hour Ago</h3>
  <TimestampInfo value={oneHourAgo} displayType="utc" />
</div>
{/* One Day Ago */}
<div className="flex flex-col gap-2 text-center">
  <h3 className="text-sm font-medium">One Day Ago</h3>
  <TimestampInfo value={oneDayAgo} displayType="utc" />
</div>
{/* One Week Ago */}
<div className="flex flex-col gap-2 text-center">
  <h3 className="text-sm font-medium">One Week Ago</h3>
  <TimestampInfo value={oneWeekAgo} displayType="utc" />
</div>`}
    >
      <Row>
        {/* Current Time */}
        <div className="flex flex-col gap-2 text-center">
          <h3 className="text-sm font-medium">Current Time</h3>
          <TimestampInfo value={now} displayType="utc" />
        </div>
        {/* One Hour Ago */}
        <div className="flex flex-col gap-2 text-center">
          <h3 className="text-sm font-medium">One Hour Ago</h3>
          <TimestampInfo value={oneHourAgo} displayType="utc" />
        </div>
        {/* One Day Ago */}
        <div className="flex flex-col gap-2 text-center">
          <h3 className="text-sm font-medium">One Day Ago</h3>
          <TimestampInfo value={oneDayAgo} displayType="utc" />
        </div>
        {/* One Week Ago */}
        <div className="flex flex-col gap-2 text-center">
          <h3 className="text-sm font-medium">One Week Ago</h3>
          <TimestampInfo value={oneWeekAgo} displayType="utc" />
        </div>
      </Row>
    </RenderComponentWithSnippet>
  );
};

export const TimestampExampleRelative = () => {
  const now = Date.now();
  const oneHourAgo = now - 3600 * 1000;
  const oneDayAgo = now - 86400 * 1000;
  const oneWeekAgo = now - 604800 * 1000;

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`{/* Current Time */}
<div className="flex flex-col gap-2 text-center">
  <h3 className="text-sm font-medium">Current Time</h3>
  <TimestampInfo value={now} displayType="relative" />
</div>
{/* One Hour Ago */}
<div className="flex flex-col gap-2 text-center">
  <h3 className="text-sm font-medium">One Hour Ago</h3>
  <TimestampInfo value={oneHourAgo} displayType="relative" />
</div>
{/* One Day Ago */}
<div className="flex flex-col gap-2 text-center">
  <h3 className="text-sm font-medium">One Day Ago</h3>
  <TimestampInfo value={oneDayAgo} displayType="relative" />
</div>
{/* One Week Ago */}
<div className="flex flex-col gap-2 text-center">
  <h3 className="text-sm font-medium">One Week Ago</h3>
  <TimestampInfo value={oneWeekAgo} displayType="relative" />
</div>`}
    >
      <Row>
        {/* Current Time */}
        <div className="flex flex-col gap-2 text-center">
          <h3 className="text-sm font-medium">Current Time</h3>
          <TimestampInfo value={now} displayType="relative" />
        </div>
        {/* One Hour Ago */}
        <div className="flex flex-col gap-2 text-center">
          <h3 className="text-sm font-medium">One Hour Ago</h3>
          <TimestampInfo value={oneHourAgo} displayType="relative" />
        </div>
        {/* One Day Ago */}
        <div className="flex flex-col gap-2 text-center">
          <h3 className="text-sm font-medium">One Day Ago</h3>
          <TimestampInfo value={oneDayAgo} displayType="relative" />
        </div>
        {/* One Week Ago */}
        <div className="flex flex-col gap-2 text-center">
          <h3 className="text-sm font-medium">One Week Ago</h3>
          <TimestampInfo value={oneWeekAgo} displayType="relative" />
        </div>
      </Row>
    </RenderComponentWithSnippet>
  );
};
