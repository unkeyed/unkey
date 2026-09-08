import { AlertTriangle } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";
import { Button } from "~/components/ui/button";

export function KeysError({ message, onRetry }: { message?: string; onRetry: () => void }) {
  return (
    <div className="flex flex-col items-center gap-4 px-5 py-12">
      <Alert variant="destructive" className="max-w-md">
        <AlertTriangle />
        <AlertTitle>Couldn't load your keys</AlertTitle>
        <AlertDescription>{message ?? "Something went wrong. Please try again."}</AlertDescription>
      </Alert>
      <Button variant="outline" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}
