import { Card, CardContent } from "@/components/ui/card";
import { ScrollText } from "lucide-react";

export const EmptyState = ({ content }: { content?: React.ReactNode }) => (
  <div className="flex-1 flex items-center justify-center">
    {content || (
      <Card className="w-96 bg-background-subtle">
        <CardContent className="flex justify-center gap-2 items-center p-6">
          <ScrollText className="h-4 w-4" />
          <div className="text-sm text-accent-12">No data available</div>
        </CardContent>
      </Card>
    )}
  </div>
);
