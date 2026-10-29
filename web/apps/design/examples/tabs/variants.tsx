import { Tabs, TabsList, TabsTrigger } from "@unkey/ui";

export default function TabsVariants() {
  return (
    <div className="flex w-full flex-col gap-6">
      <Tabs defaultValue="traffic">
        <TabsList>
          <TabsTrigger value="traffic">Traffic</TabsTrigger>
          <TabsTrigger value="resources">Resources</TabsTrigger>
          <TabsTrigger value="instances">Instances</TabsTrigger>
        </TabsList>
      </Tabs>
      <Tabs defaultValue="traffic">
        <TabsList variant="underline">
          <TabsTrigger value="traffic">Traffic</TabsTrigger>
          <TabsTrigger value="resources">Resources</TabsTrigger>
          <TabsTrigger value="instances">Instances</TabsTrigger>
        </TabsList>
      </Tabs>
    </div>
  );
}
