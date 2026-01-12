import { Card, CardContent, CopyButton } from "@unkey/ui";

export const LogSection = ({
  details,
  title,
}: {
  details: string | string[];
  title: string;
}) => {
  return (
    <div className="flex flex-col gap-1 mt-[16px]">
      <div className="flex justify-between items-center">
        <span className="text-[13px] text-accent-9 font-sans">{title}</span>
      </div>
      <Card className="bg-gray-2 border-gray-4 rounded-lg">
        <CardContent className="py-2 px-3 text-xs relative group ">
          <pre className="flex flex-col gap-1 whitespace-pre-wrap leading-relaxed">
            {Array.isArray(details)
              ? details
                  .slice()
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
                      <div className="group flex items-center w-full p-[3px]" key={uniqueKey}>
                        <span className="w-28 text-left truncate text-accent-9">{key}:</span>
                        <span className="ml-2 text-xs text-accent-12 secret">{value}</span>
                      </div>
                    );
                  })
              : details}
          </pre>
          <CopyButton
            value={getFormattedContent(details)}
            shape="square"
            variant="primary"
            size="2xlg"
            className="absolute bottom-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity rounded-md p-4 secret"
            aria-label="Copy content"
          />
        </CardContent>
      </Card>
    </div>
  );
};

const getFormattedContent = (details: string | string[]) => {
  if (Array.isArray(details)) {
    return details
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
  return details;
};
