import { CopyButton } from "@unkey/ui";

export const LogSection = ({
  details,
  title,
}: {
  details: Record<string, React.ReactNode> | string;
  title: string;
}) => {
  const getTextToCopy = () => {
    if (typeof details === "string") {
      return details;
    }

    return Object.entries(details)
      .map(([key, value]) => {
        if (value === null || value === undefined) {
          return key;
        }
        if (typeof value === "object" && value !== null && "props" in value) {
          return `${key}: ${value}`;
        }
        return `${key}: ${value}`;
      })
      .join("\n");
  };

  return (
    <div className="flex flex-col gap-1 mt-[16px] px-4">
      <div className="border bg-gray-2 border-gray-4 rounded-[10px] relative group">
        <div className="text-gray-11 text-[12px] leading-6 px-[14px] py-1.5 font-sans">{title}</div>
        <div className="border-gray-4 border-t rounded-[10px] bg-white dark:bg-black px-3.5 py-2">
          <pre className="whitespace-pre-wrap break-words leading-relaxed text-xs">
            {typeof details === "object"
              ? Object.entries(details).map(([key, value]) => (
                  <div className="flex items-center w-full px-[3px] leading-7" key={key}>
                    <span className="text-left text-gray-11 whitespace-nowrap">
                      {key}
                      {value ? ":" : ""}
                    </span>
                    <span className="ml-2 text-accent-12 truncate">{value}</span>
                  </div>
                ))
              : details}
          </pre>
        </div>
        <CopyButton
          value={getTextToCopy()}
          shape="square"
          variant="outline"
          className="absolute bottom-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity rounded-md p-4 bg-gray-2 hover:bg-gray-2 size-2"
          aria-label="Copy content"
        />
      </div>
    </div>
  );
};
