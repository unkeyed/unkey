import { IconBookmarkFill18 } from "nucleo-ui-fill-18";
import { IconBookmarkOutline18 } from "nucleo-ui-outline-18";

export default function IconFillAndOutline() {
  return (
    <div className="flex items-end gap-10">
      <div className="flex flex-col items-center gap-3">
        <IconBookmarkOutline18 className="text-gray-11" />
        <span className="text-gray-10 text-xs">Not bookmarked</span>
      </div>
      <div className="flex flex-col items-center gap-3">
        <IconBookmarkFill18 className="text-accent-12" />
        <span className="text-gray-10 text-xs">Bookmarked</span>
      </div>
    </div>
  );
}
