"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Button } from "@unkey/ui";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@unkey/ui";

export function BasicCard() {
  return (
    <RenderComponentWithSnippet>
      <div className="w-full max-w-md">
        <Card>
          <CardContent>
            <p>
              This is a basic card with just content. Perfect for simple layouts where you only need
              a container with consistent styling.
            </p>
          </CardContent>
        </Card>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CardWithHeader() {
  return (
    <RenderComponentWithSnippet>
      <div className="w-full max-w-md">
        <Card>
          <CardHeader>
            <CardTitle>Card Title</CardTitle>
            <CardDescription>
              This card has a header with a title and description. Great for introducing content
              sections.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <p>
              Your main content goes here. The header provides clear context for what this card
              contains.
            </p>
          </CardContent>
        </Card>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CardWithFooter() {
  return (
    <RenderComponentWithSnippet>
      <div className="w-full max-w-md">
        <Card>
          <CardContent>
            <p>
              This card includes a footer section with actions. The footer has a border separator
              and consistent spacing.
            </p>
          </CardContent>
          <CardFooter>
            <Button variant="outline" className="mr-2">
              Cancel
            </Button>
            <Button>Save</Button>
          </CardFooter>
        </Card>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CompleteCard() {
  return (
    <RenderComponentWithSnippet>
      <div className="w-full max-w-md">
        <Card>
          <CardHeader>
            <CardTitle>Complete Card Example</CardTitle>
            <CardDescription>
              This card demonstrates all components working together: header, content, and footer.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <p>This is the main content area where you can place any content you need.</p>
              <div className="grid grid-cols-2 gap-4">
                <div className="p-2 bg-gray-50 rounded">Item 1</div>
                <div className="p-2 bg-gray-50 rounded">Item 2</div>
              </div>
            </div>
          </CardContent>
          <CardFooter>
            <div className="flex justify-between w-full">
              <span className="text-sm text-gray-500">Last updated: 2 hours ago</span>
              <div>
                <Button variant="outline" className="mr-2">
                  Edit
                </Button>
                <Button>View Details</Button>
              </div>
            </div>
          </CardFooter>
        </Card>
      </div>
    </RenderComponentWithSnippet>
  );
}

export function CardGrid() {
  return (
    <RenderComponentWithSnippet>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 w-full">
        <Card>
          <CardHeader>
            <CardTitle>API Keys</CardTitle>
            <CardDescription>Manage your API keys</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm">Create and manage API keys for your applications.</p>
          </CardContent>
          <CardFooter>
            <Button className="w-full">Manage Keys</Button>
          </CardFooter>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Analytics</CardTitle>
            <CardDescription>View usage analytics</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div className="flex justify-between">
                <span className="text-sm">Requests</span>
                <span>45.2K</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm">Success Rate</span>
                <span>99.9%</span>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Settings</CardTitle>
            <CardDescription>Configure your workspace</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm">Update your workspace settings and preferences.</p>
          </CardContent>
          <CardFooter>
            <Button variant="outline" className="w-full">
              Configure
            </Button>
          </CardFooter>
        </Card>
      </div>
    </RenderComponentWithSnippet>
  );
}
