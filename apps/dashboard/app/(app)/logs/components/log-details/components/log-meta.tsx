import { Button } from "@unkey/ui";
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
    <div className="flex justify-between pt-2.5 px-3">
      <div className="text-sm text-content/65 font-sans">Meta</div>
      <Card className="rounded-[5px] flex">
        <CardContent className="text-[12px] w-[300px] flex-2 bg-background-subtle p-3 rounded-[5px] relative group">
          <pre>{content}</pre>
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
