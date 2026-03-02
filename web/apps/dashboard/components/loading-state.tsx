import { Loading } from "@unkey/ui";

export function LoadingState({ message = "Loading..." }: { message?: string }) {
  return (
    <div className="h-[100dvh] relative flex flex-col overflow-hidden bg-white dark:bg-base-12 lg:flex-row">
      <div className="flex items-center justify-center w-full h-full">
        <div className="flex flex-col items-center gap-4">
          <Loading size={24} />
          <p className="text-sm text-gray-600 dark:text-gray-400">{message}</p>
        </div>
      </div>
    </div>
  );
}
