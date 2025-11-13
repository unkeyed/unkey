import { Loading } from "@unkey/ui";

type Props = {
  message?: string;
  spinnerType?: "dots" | "spinner";
  size?: number;
  className?: string;
};

const PageLoading = ({
  message = "Loading...",
  spinnerType = "spinner",
  size = 24,
  className,
}: Props) => {
  return (
    <div className={`flex items-center justify-center w-full h-full min-h-[600px] ${className}`}>
      <div className="flex flex-col items-center gap-4">
        <Loading size={size} type={spinnerType} />
        <p className="text-sm text-gray-600 dark:text-gray-400">{message}</p>
      </div>
    </div>
  );
};

export { PageLoading };
