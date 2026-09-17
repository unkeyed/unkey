import { formatTimeSinceLastUse } from "@/lib/recently-used-key";
import { IconTriangleWarningOutline12 } from "@unkey/icons";

export const RecentlyUsedKeyWarning = ({ lastUsedAt }: { lastUsedAt: number }) => {
  return (
    <div className="rounded-xl bg-warningA-2 dark:bg-black border border-warningA-6 flex items-center gap-4 px-[22px] py-6 mt-2">
      <div className="bg-warning-9 size-8 rounded-full flex items-center justify-center shrink-0">
        <IconTriangleWarningOutline12 className="text-white" />
      </div>
      <div className="text-gray-12 text-[13px] leading-6">
        <span className="font-medium">Still in use:</span> this key was last used{" "}
        {formatTimeSinceLastUse(lastUsedAt)}. Deleting it will immediately break anything that is
        still authenticating with it.
      </div>
    </div>
  );
};
