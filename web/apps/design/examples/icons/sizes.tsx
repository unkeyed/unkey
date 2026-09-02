import { IconGearOutline12 } from "nucleo-ui-outline-12";
import { IconGearOutline18 } from "nucleo-ui-outline-18";

export default function IconSizes() {
  return (
    <div className="flex items-end gap-10 text-accent-12">
      <div className="flex flex-col items-center gap-3">
        <IconGearOutline12 />
        <span className="text-gray-10 text-xs">12px</span>
      </div>
      <div className="flex flex-col items-center gap-3">
        <IconGearOutline18 className="size-4" />
        <span className="text-gray-10 text-xs">16px</span>
      </div>
      <div className="flex flex-col items-center gap-3">
        <IconGearOutline18 />
        <span className="text-gray-10 text-xs">18px</span>
      </div>
    </div>
  );
}
