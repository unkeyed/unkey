import { Card } from "@/components/ui/card";
import { Loader2 } from "lucide-react";

export default function Loading() {
  return (
    <Card className="flex items-center justify-center w-full h-96">
      <Loader2 className="w-6 h-6 animate-spin" />
    </Card>
  );
}
