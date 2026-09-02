import { IconCubeOutline18 } from "nucleo-ui-outline-18";

export default function IconHero() {
  return (
    <div className="flex items-center gap-10 text-accent-12">
      <IconCubeOutline18 className="size-9 [&_[stroke-width]]:[stroke-width:0.75]" />
      <div className="flex size-16 items-center justify-center rounded-2xl border border-gray-4 bg-gray-2 [&_svg]:size-7 [&_svg_[stroke-width]]:[stroke-width:1]">
        <IconCubeOutline18 />
      </div>
    </div>
  );
}
