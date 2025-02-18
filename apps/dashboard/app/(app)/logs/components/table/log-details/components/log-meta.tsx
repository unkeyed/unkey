import { Card, CardContent } from "@/components/ui/card";
import { toast } from "@/components/ui/toaster";
import { Button } from "@unkey/ui";
import { Copy } from "lucide-react";

export const LogMetaSection = ({ content }: { content: string }) => {
  const handleClick = () => {
    navigator.clipboard
      .writeText(content)
      .then(() => {
        toast.success("Meta copied to clipboard");
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
  };

  return (
    <div className="flex justify-between pt-2.5">
      <div className="text-[13px] text-accent-9 font-sans">Meta</div>
      <Card className="bg-gray-2 border-gray-4 rounded-lg">
        <CardContent className="py-2 px-3 text-xs relative group min-w-[300px]">
          <pre className="text-accent-12">{content ?? "<EMPTY>"}</pre>
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
