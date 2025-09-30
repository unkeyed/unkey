import { Card, CardContent, CopyButton } from "@unkey/ui";

export const LogMetaSection = ({ content }: { content: string }) => {
  return (
    <div className="flex justify-between pt-2.5 flex-col gap-1">
      <div className="text-[13px] text-accent-11 font-sans">Meta</div>
      <Card className="bg-gray-2 border-gray-4 rounded-lg">
        <CardContent className="py-2 px-3 text-xs relative group min-w-[300px]">
          <pre className="text-accent-12">{content ?? "<EMPTY>"} </pre>
          <CopyButton
            value={content}
            shape="square"
            variant="primary"
            size="2xlg"
            className="absolute bottom-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity rounded-md p-4"
            aria-label="Copy content"
          />
        </CardContent>
      </Card>
    </div>
  );
};
