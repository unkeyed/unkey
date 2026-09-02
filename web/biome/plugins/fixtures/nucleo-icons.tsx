// Fixture for the Nucleo icon plugins. Every element in the first three
// components must be reported and nothing in Allowed may be; run
// `pnpm lint:icons` to check that the plugins still fire.

export function LockedStrokeAndSize() {
  return (
    <div>
      <IconPlusOutline18 strokeWidth={2} />
      <IconPlusOutline12 size={12} />
      <IconBookmarkFill18 width={18} height={18} />
      <IconEarthOutline18 size="18" />
      <IconGridOutline18 strokeWidth={1}>label</IconGridOutline18>
    </div>
  );
}

export function BoxAndStrokeClasses() {
  return (
    <div>
      <IconEarthOutline18 className="w-4 h-4" />
      <IconEarthOutline18 className="text-gray-9 h-4 w-4 shrink-0" />
      <IconPlusOutline12 className="w-3" />
      <IconBookmarkFill18 className="size-4 w-full" />
      <IconCubeOutline18 className="size-9 [&_[stroke-width]]:[stroke-width:0.75]" />
    </div>
  );
}

export function EighteenDrawnTooSmallWithTwelveAvailable() {
  return (
    <div>
      <IconPlusOutline18 className="size-3" />
      <IconGearOutline18 className="text-gray-9 size-2.5" />
      <IconGearOutline18 className="size-[12px] text-gray-9" />
      <IconPlusOutline18 className="shrink-0 size-1.5">label</IconPlusOutline18>
    </div>
  );
}

export function Allowed() {
  return (
    <div>
      <IconEarthOutline18 className="size-4" />
      <IconEarthOutline18 className="size-30" />
      <IconPlusOutline12 className="size-3" />
      <IconEarthOutline18 className="size-3" />
      <IconCubeOutline18 className="size-3 max-h-4 min-w-3" />
      <IconPlusOutline18 className="size-3.5 hover:opacity-80" />
      <Button size="md" width={3} className="size-3 h-4 w-4" />
      <div className="[&_svg_[stroke-width]]:[stroke-width:0.75] w-4 h-4" />
    </div>
  );
}
