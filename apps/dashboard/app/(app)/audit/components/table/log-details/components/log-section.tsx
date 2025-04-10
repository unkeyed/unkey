import { Card, CardContent } from "@/components/ui/card";
import { toast } from "@/components/ui/toaster";
import { Button } from "@unkey/ui";
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
      <div className="flex items-center justify-between">
        <span className="text-[13px] text-accent-9 font-sans">{title}</span>
      </div>
      <Card className="rounded-lg bg-gray-2 border-gray-4">
        <CardContent className="relative px-3 py-2 text-xs group ">
          <pre className="flex flex-col gap-1 leading-relaxed whitespace-pre-wrap">
            {Array.isArray(details)
              ? details.map((header) => {
                const [key, ...valueParts] = header.split(":");
                const value = valueParts.join(":").trim();
                return (
                  <div className="group flex items-center w-full p-[3px]" key={key}>
                    <span className="text-left truncate w-28 text-accent-9">{key}:</span>
                    <span className="ml-2 text-xs text-accent-12 ">{value}</span>
                  </div>
                );
              })
              : details}
          </pre>
          <Button
            shape="square"
            onClick={handleClick}
            className="absolute transition-opacity opacity-0 bottom-2 right-3 group-hover:opacity-100"
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