import { AlertTriangle } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "~/components/ui/alert";
import { Button } from "~/components/ui/button";

export function SessionExpired({ returnUrl }: { returnUrl: string | null }) {
  return (
    <div className="flex flex-col items-center gap-4">
      <Alert className="max-w-md">
        <AlertTriangle />
        <AlertTitle>Your session has expired</AlertTitle>
        <AlertDescription>
          Return to your application to continue managing your API keys.
        </AlertDescription>
      </Alert>
      {returnUrl && (
        <Button variant="outline" render={<a href={returnUrl}>Back to application</a>} />
      )}
    </div>
  );
}
