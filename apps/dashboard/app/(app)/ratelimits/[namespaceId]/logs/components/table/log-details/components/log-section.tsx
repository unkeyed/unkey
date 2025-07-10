import { Button, Card, CardContent, toast } from "@unkey/ui";
import { Copy } from "lucide-react";

export const LogSection = ({
  details,
  title,
}: {
  details: string | string[];
  title: string;
}) => {
  const handleClick = () => {
    navigator.clipboard
      .writeText(getFormattedContent(details))
      .then(() => {
        toast.success(`${title} copied to clipboard`);
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
  };

  return (
    <div className="flex flex-col gap-1 mt-[16px]">
      <div className="flex justify-between items-center">
        <span className="text-[13px] text-accent-9 font-sans">{title}</span>
      </div>
      <Card className="bg-gray-2 border-gray-4 rounded-lg">
        <CardContent className="py-2 px-3 text-xs relative group ">
          <pre className="flex flex-col gap-1 whitespace-pre-wrap leading-relaxed">
            {Array.isArray(details)
              ? details.map((header) => {
                  const [key, ...valueParts] = header.split(":");
                  const value = valueParts.join(":").trim();
                  return (
                    <div className="group flex items-center w-full p-[3px]" key={key}>
                      <span className="text-left text-accent-9 whitespace-nowrap">{key}:</span>
                      <span className="ml-2 text-xs text-accent-12 truncate">{value}</span>
                    </div>
                  );
                })
              : details}
          </pre>
          <Button
            shape="square"
            onClick={handleClick}
            className="absolute bottom-2 right-3 opacity-0 group-hover:opacity-100 transition-opacity"
            aria-label="Copy content"
          >
            <Copy size={14} />
          </Button>
        </CardContent>
      </Card>
    </div>
  );
};

const getFormattedContent = (details: string | string[]) => {
  if (Array.isArray(details)) {
    return details
      .map((header) => {
        const [key, ...valueParts] = header.split(":");
        const value = valueParts.join(":").trim();
        return `${key}: ${value}`;
      })
      .join("\n");
  }
  return details;
};
