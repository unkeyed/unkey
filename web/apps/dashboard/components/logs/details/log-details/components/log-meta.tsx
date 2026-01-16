import { CopyButton } from "@unkey/ui";

export const LogMetaSection = ({ content }: { content: React.ReactNode }) => {
  return (
    <div className="flex flex-col gap-1 mt-[16px] px-4">
      <div className="border bg-gray-2 border-gray-4 rounded-[10px] relative group">
        <div className="text-gray-11 text-[12px] leading-6 px-[14px] py-1.5 font-sans">Meta</div>
        <div className="border-gray-4 border-t rounded-[10px] bg-white dark:bg-black px-3.5 py-2">
          <pre className="whitespace-pre-wrap leading-relaxed text-xs text-accent-12">
            {content}
          </pre>
        </div>
        <CopyButton
          value={typeof content === "string" ? content : ""}
          shape="square"
          variant="outline"
          className="absolute bottom-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity rounded-md p-4 bg-gray-2 hover:bg-gray-2 size-2"
          aria-label="Copy content"
        />
      </div>
    </div>
  );
};
