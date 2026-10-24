import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@unkey/ui";

export default function BasicCard() {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">checkout-api</CardTitle>
        <CardDescription>2M requests this week</CardDescription>
      </CardHeader>
      <CardContent className="text-gray-11 text-sm">
        Live in us-east-1 and eu-central-1.
      </CardContent>
    </Card>
  );
}
