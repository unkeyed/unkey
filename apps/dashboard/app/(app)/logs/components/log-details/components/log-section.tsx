import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { toast } from "@/components/ui/toaster";
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
    <div className="px-3 flex flex-col gap-[2px]">
      <div className="flex justify-between items-center">
        <span className="text-sm text-content/65 font-sans">{title}</span>
      </div>
      <Card className="rounded-[5px] bg-background-subtle">
        <CardContent className="p-2 text-[12px] relative group">
          <pre>
            {Array.isArray(details)
              ? details.map((header) => {
                  const [key, ...valueParts] = header.split(":");
                  const value = valueParts.join(":").trim();
                  return (
                    <span key={header}>
                      <span className="text-content/65">{key}</span>
                      <span className="text-content whitespace-pre-line">: {value}</span>
                      {"\n"}
                    </span>
                  );
                })
              : details}
          </pre>
          <Button
            variant="outline"
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
