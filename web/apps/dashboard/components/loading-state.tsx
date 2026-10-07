import { Loading } from "@unkey/ui";

type LoadingStateProps = {
  delayMs?: number;
  message?: string;
};

export function LoadingState({ delayMs, message = "Loading..." }: LoadingStateProps) {
  return (
    <div className="flex-1 relative flex flex-col overflow-hidden bg-background lg:flex-row">
      <div className="flex items-center justify-center w-full flex-1">
        <div
          className={
            delayMs === undefined
              ? "flex flex-col items-center gap-4"
              : "flex flex-col items-center gap-4 animate-in fade-in duration-200 [animation-fill-mode:backwards] motion-reduce:animate-none"
          }
          style={delayMs === undefined ? undefined : { animationDelay: `${delayMs}ms` }}
        >
          <Loading size={24} />
          <p className="text-sm text-gray-11">{message}</p>
        </div>
      </div>
    </div>
  );
}
