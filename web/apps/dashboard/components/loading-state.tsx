import { Loading } from "@unkey/ui";

export function LoadingState({ message = "Loading..." }: { message?: string }) {
  return (
    <div className="flex-1 relative flex flex-col overflow-hidden bg-white dark:bg-base-12 lg:flex-row">
      <div className="flex items-center justify-center w-full flex-1">
        <div className="flex flex-col items-center gap-4">
          <Loading size={24} />
          <p className="text-sm text-gray-11">{message}</p>
        </div>
      </div>
    </div>
  );
}
