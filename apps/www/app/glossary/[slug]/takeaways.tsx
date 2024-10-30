import type { Glossary } from "@/.content-collections/generated";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import {
  AlertTriangle,
  BookOpen,
  Clock,
  Code,
  Coffee,
  ExternalLink,
  FileText,
  RefreshCcw,
  Zap,
} from "lucide-react";
import { z } from "zod";

const itemSchema = z.object({
  key: z.string(),
  value: z.string(),
});

export const takeawaysSchema = z.object({
  tldr: z.string(),
  definitionAndStructure: z.array(itemSchema),
  historicalContext: z.array(itemSchema),
  usageInAPIs: z.object({
    tags: z.array(z.string()),
    description: z.string(),
  }),
  bestPractices: z.array(z.string()),
  recommendedReading: z.array(
    z.object({
      title: z.string(),
      url: z.string(),
    }),
  ),
  didYouKnow: z.string(),
});

export default function Takeaways(props: Pick<Glossary, "term" | "takeaways">) {
  return (
    <Card className="w-full bg-white/5 shadow-[0_0_10px_rgba(255,255,255,0.1)] rounded-xl overflow-hidden relative border-white/20">
      <div className="absolute left-0 top-0 bottom-0 w-1 bg-white/20" />
      <CardHeader className="border-white/20">
        <CardTitle className="text-2xl font-bold text-white">{props.term}: Key Takeaways</CardTitle>
      </CardHeader>
      <CardContent className="space-y-8 p-6">
        <div className="bg-white/10 p-4 rounded-md">
          <h3 className="text-lg font-semibold flex items-center mb-2 text-white">
            <Zap className="mr-2 h-5 w-5" /> TL;DR
          </h3>
          <p className="text-sm text-white/80">{props.takeaways.tldr}</p>
        </div>
        <div className="grid gap-8 md:grid-cols-2">
          <Section
            icon={<FileText className="h-5 w-5" />}
            title="Definition & Structure"
            content={
              <div className="space-y-2">
                {props.takeaways.definitionAndStructure.map((item) => (
                  <div key={item.key} className="flex justify-between text-sm">
                    <span className="font-medium text-white/80">{item.key}</span>
                    <code className="bg-white/10 px-1 py-0.5 rounded tracking-tight text-xs text-white/90">
                      {item.value}
                    </code>
                  </div>
                ))}
              </div>
            }
          />
          <Section
            icon={<Clock className="h-5 w-5" />}
            title="Historical Context"
            items={props.takeaways.historicalContext}
          />
          <Section
            icon={<Code className="h-5 w-5" />}
            title="Usage in APIs"
            content={
              <>
                <div className="flex flex-wrap gap-1 mb-2">
                  {props.takeaways.usageInAPIs.tags.map((tag) => (
                    <Badge key={tag} variant="secondary" className="bg-white/10 text-white/80">
                      {tag}
                    </Badge>
                  ))}
                </div>
                <p className="text-sm text-white/60">{props.takeaways.usageInAPIs.description}</p>
              </>
            }
          />
          <Section
            icon={<AlertTriangle className="h-5 w-5" />}
            title="Best Practices"
            items={props.takeaways.bestPractices}
          />
        </div>
        <Section
          icon={<BookOpen className="h-5 w-5" />}
          title="Recommended Reading"
          content={
            <ul className="space-y-2 text-sm">
              {props.takeaways.recommendedReading.map((item) => (
                <li key={item.title}>
                  <a
                    href={item.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-300 flex items-center"
                  >
                    {item.title}
                    <ExternalLink className="h-3 w-3 ml-1" />
                  </a>
                </li>
              ))}
            </ul>
          }
        />
      </CardContent>
      <CardFooter className="border-t text-xs border-white/10">
        <div className="grid grid-cols-[auto_1fr_auto] gap-4 items-center w-full mt-6">
          <div className="flex items-center">
            <Coffee className="size-5 text-white/80" />
            <span className="font-semibold text-white ml-2">Did You Know?</span>
          </div>
          <span className="text-white/60">{props.takeaways.didYouKnow}</span>
          <RefreshCcw className="h-4 w-4 text-white/60 cursor-pointer" />
        </div>
      </CardFooter>
    </Card>
  );
}

type SectionProps = {
  icon: React.ReactNode;
  title: string;
} & (
  | { items: Array<string | z.infer<typeof itemSchema>>; content?: never }
  | { items?: never; content: React.ReactNode }
);

function Section(props: SectionProps) {
  const { icon, title } = props;
  return (
    <div className="space-y-2">
      <h3 className="text-lg font-semibold flex items-center text-white">
        {icon}
        <span className="ml-2">{title}</span>
      </h3>
      {props.content ? (
        props.content
      ) : (
        <div>
          {props.items?.map((item) =>
            typeof item === "string" ? (
              <li key={item} className="text-sm ml-4 text-white/60">
                {item}
              </li>
            ) : (
              <div key={item.key} className="flex justify-between text-sm space-y-2">
                <span className="font-medium text-white/80">{item.key}</span>
                <span className="text-white/60">{item.value}</span>
              </div>
            ),
          )}
        </div>
      )}
    </div>
  );
}
