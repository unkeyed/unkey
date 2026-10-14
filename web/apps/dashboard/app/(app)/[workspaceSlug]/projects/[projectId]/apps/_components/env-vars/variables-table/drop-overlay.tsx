import { IconCloudUploadOutline18 } from "@unkey/icons";
import { cn } from "cn";

export function DropOverlay({ isDragging }: { isDragging: boolean }) {
  return (
    <div
      className={cn(
        "pointer-events-none absolute inset-0 z-10 flex items-center justify-center transition-all duration-200",
        isDragging ? "bg-successA-2 opacity-100" : "opacity-0",
      )}
    >
      <div
        className={cn(
          "absolute inset-2 rounded-md border-2 border-dashed transition-all duration-200",
          isDragging ? "scale-100 border-successA-8" : "scale-[0.98] border-transparent",
        )}
      />
      <div
        className={cn(
          "flex items-center gap-3 transition-all duration-200",
          isDragging ? "scale-100 opacity-100" : "scale-95 opacity-0",
        )}
      >
        <div className="flex size-8 items-center justify-center rounded-lg bg-successA-3">
          <IconCloudUploadOutline18 className="text-success-11" />
        </div>
        <span className="text-sm font-medium text-success-11">Drop your .env file</span>
      </div>
    </div>
  );
}
