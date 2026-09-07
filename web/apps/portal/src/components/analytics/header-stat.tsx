type Props = {
  label: string;
  value: string | null;
  swatch?: string;
};

export function HeaderStat({ label, value, swatch }: Props) {
  return (
    <div className="flex flex-col gap-1.5 px-5 py-4 sm:px-6 sm:py-5">
      <span className="flex items-center gap-2 text-gray-11 text-sm">
        {swatch && (
          <span className="size-2 rounded-xs" style={{ backgroundColor: swatch }} aria-hidden />
        )}
        {label}
      </span>
      {value === null ? (
        <div className="h-8 w-28 rounded bg-gray-3 motion-safe:animate-pulse sm:h-9" />
      ) : (
        <span className="font-semibold text-2xl text-gray-12 tabular-nums tracking-tight sm:text-3xl">
          {value}
        </span>
      )}
    </div>
  );
}
