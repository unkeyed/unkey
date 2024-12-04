import { Card, CardContent } from "@/components/ui/card";

export const LogSection = ({
  details,
  title,
}: {
  details: string[];
  title: string;
}) => {
  return (
    <div className="px-3 flex flex-col gap-[2px]">
      <span className="text-sm text-content/65 font-sans">{title}</span>
      <Card className="rounded-[5px] bg-background-subtle">
        <CardContent className="p-2 whitespace-pre-wrap text-[12px]">
          <pre className="font-mono text-[12px] text-gray-600 whitespace-pre">
            {details.map((header) => {
              const [key, ...valueParts] = header.split(":");
              const value = valueParts.join(":").trim();
              return (
                <span key={header}>
                  <span className="text-content/65 capitalize">{key}</span>
                  <span className="text-content whitespace-pre-line">
                    : {value}
                  </span>
                  {"\n"}
                </span>
              );
            })}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
};
