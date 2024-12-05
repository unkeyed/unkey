import { Card, CardContent } from "@/components/ui/card";

export const LogMetaSection = ({ content }: { content: string }) => {
  return (
    <div className="flex justify-between pt-2.5 px-3">
      <div className="text-sm text-content/65 font-sans">Meta</div>
      <Card className="rounded-[5px] flex">
        <CardContent className="text-[12px] w-[300px] flex-2 bg-background-subtle p-3 rounded-[5px]">
          <pre>{content}</pre>
        </CardContent>
      </Card>
    </div>
  );
};
