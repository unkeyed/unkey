import { TOP_NAV_HEIGHT } from "@/components/navigation/top-nav";
import { formatStageDuration } from "./panes/deploying/run-model";

export const COLUMN_GAP = "clamp(24px, 4vh, 48px)";
export const COLUMN_TOP_SPACE = "clamp(40px, 8vh, 120px)";
const MIN_BOTTOM_PEEK_PX = 40;
export const COLUMN_CARD_MAX_HEIGHT = `min(640px, calc(100dvh - ${COLUMN_TOP_SPACE} - 2 * ${COLUMN_GAP} - ${TOP_NAV_HEIGHT + MIN_BOTTOM_PEEK_PX}px))`;

export const cardFooter =
  "flex shrink-0 items-center gap-3 border-t border-grayA-4 bg-grayA-2 px-5 py-3";

export const cardSurface = "flex min-h-0 flex-col overflow-hidden rounded-lg border bg-raised";

export function CardTitle({ title, description }: { title: string; description: string }) {
  return (
    <div className="flex flex-col gap-1">
      <h1 className="text-lg font-semibold leading-7 text-gray-12">{title}</h1>
      <p className="text-sm leading-5 text-gray-11">{description}</p>
    </div>
  );
}

export function CardHeader({
  title,
  description,
  elapsedMs,
}: {
  title: string;
  description: string;
  elapsedMs: number | null;
}) {
  return (
    <div className="flex shrink-0 items-start gap-4">
      <CardTitle title={title} description={description} />
      <span className="ml-auto font-mono text-sm tabular-nums text-gray-11">
        {elapsedMs === null ? "" : formatStageDuration(elapsedMs)}
      </span>
    </div>
  );
}
