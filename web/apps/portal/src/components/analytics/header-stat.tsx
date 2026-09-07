type Props = {
  label: string;
  value: string | null;
  swatch?: string;
};

export function HeaderStat({ label, value, swatch }: Props) {
  return (
    <div className="flex flex-col gap-1">
      <span className="flex items-center gap-1.5 text-gray-11 text-xs">
        {swatch && (
          <span className="size-2 rounded-xs" style={{ backgroundColor: swatch }} aria-hidden />
        )}
        {label}
      </span>
      {value === null ? (
        <div className="h-6 w-20 rounded bg-gray-3 motion-safe:animate-pulse" />
      ) : (
        <span className="font-semibold text-base text-gray-12 tabular-nums">{value}</span>
      )}
    </div>
  );
}
