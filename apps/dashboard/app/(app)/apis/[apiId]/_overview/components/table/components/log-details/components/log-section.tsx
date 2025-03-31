"use client";
import { Card, CardContent } from "@/components/ui/card";
import { toast } from "@/components/ui/toaster";
import { Clone } from "@unkey/icons";
import { Button } from "@unkey/ui";

export const LogSection = ({
  details,
  title,
}: {
  details: Record<string, React.ReactNode> | string;
  title: string;
}) => {
  const handleClick = () => {
    navigator.clipboard
      .writeText(JSON.stringify(details))
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
        <CardContent className="py-2 px-3 text-xs relative group">
          <pre className="flex flex-col gap-1 whitespace-pre-wrap leading-relaxed">
            {typeof details === "object"
              ? Object.entries(details).map((detail) => {
                  const [key, value] = detail;
                  return (
                    <div className="group flex items-center w-full p-[3px]" key={key}>
                      <span className="text-left text-accent-9 whitespace-nowrap">
                        {key}
                        {value ? ":" : ""}
                      </span>
                      <span className="ml-2 text-xs text-accent-12 truncate">{value}</span>
                    </div>
                  );
                })
              : details}
          </pre>
          <Button
            shape="square"
            onClick={handleClick}
            variant="outline"
            className="absolute bottom-2 right-3 opacity-0 group-hover:opacity-100 transition-opacity rounded-sm"
            aria-label="Copy content"
          >
            <Clone />
          </Button>
        </CardContent>
      </Card>
    </div>
  );
};
