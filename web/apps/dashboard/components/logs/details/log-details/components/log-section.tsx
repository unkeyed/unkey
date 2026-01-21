import { CopyButton } from "@unkey/ui";

type LogSectionDetails = string | string[] | React.ReactNode;

export const LogSection = ({
  details,
  title,
}: {
  details: LogSectionDetails;
  title: string;
}) => {
  return (
    <div className="flex flex-col gap-1 mt-[16px] px-4">
      <div className="border bg-gray-2 border-gray-4 rounded-[10px] relative group">
        <div className="text-gray-11 text-xs leading-6 px-[14px] py-1.5 font-sans">{title}</div>
        <div className="border-gray-4 border-t rounded-[10px] bg-white dark:bg-black px-3.5 py-2">
          <pre className="whitespace-pre-wrap break-words leading-relaxed text-xs text-accent-12">
            {Array.isArray(details)
              ? [...details]
                  .sort((a, b) => {
                    const keyA = a.split(":")[0].toLowerCase();
                    const keyB = b.split(":")[0].toLowerCase();
                    return keyA.localeCompare(keyB);
                  })
                  .map((header, index) => {
                    const [key, ...valueParts] = header.split(":");
                    const value = valueParts.join(":").trim();
                    // Create unique key by combining key, value hash, and position for duplicates
                    const uniqueKey = `${key}-${value.slice(0, 20)}-${header.length}-${index}`;
                    return (
                      <div className="flex items-center w-full px-[3px] leading-7" key={uniqueKey}>
                        <span className="text-left text-gray-11 whitespace-nowrap">{key}:</span>
                        <span className="ml-2 text-accent-12 truncate">{value}</span>
                      </div>
                    );
                  })
              : details}
          </pre>
        </div>
        <CopyButton
          value={getFormattedContent(details)}
          shape="square"
          variant="outline"
          className="absolute bottom-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity rounded-md p-4 bg-gray-2 hover:bg-gray-2 size-2"
          aria-label="Copy content"
        />
      </div>
    </div>
  );
};

const getFormattedContent = (details: LogSectionDetails): string => {
  if (Array.isArray(details)) {
    return [...details]
      .sort((a, b) => {
        const keyA = a.split(":")[0].toLowerCase();
        const keyB = b.split(":")[0].toLowerCase();
        return keyA.localeCompare(keyB);
      })
      .map((header) => {
        const [key, ...valueParts] = header.split(":");
        const value = valueParts.join(":").trim();
        return `${key}: ${value}`;
      })
      .join("\n");
  }

  if (typeof details === "string") {
    return details;
  }

  return "";
};
