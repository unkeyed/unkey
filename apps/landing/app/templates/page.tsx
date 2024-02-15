"use client";
import { Container } from "@/components/container";
import { CTA } from "@/components/cta";
import { CodeIcon, FrameworkIcon, TemplatesRightArrow } from "@/components/svg/template-page";
import { Checkbox } from "@/components/template/checkbox";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/template/form";
import { SearchInput } from "@/components/template/input";
import { PageIntro } from "@/components/template/page-intro";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Separator } from "@/components/ui/separator";
import {
  TemplatesFormValues,
  getDefaulTemplatesFormValues,
  schema,
  updateUrl,
} from "@/lib/templates-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { VenetianMask } from "lucide-react";
import Link from "next/link";
import { useEffect, useMemo } from "react";
import { useForm } from "react-hook-form";
import { Framework, Language, templates } from "./data";

export default function Templates() {
  const form = useForm<TemplatesFormValues>({
    resolver: zodResolver(schema),
    defaultValues: getDefaulTemplatesFormValues(),
    reValidateMode: "onChange",
  });

  const languages = Object.values(templates).reduce(
    (acc, { language }) => {
      if (!acc[language]) {
        acc[language] = 0;
      }
      acc[language]++;
      return acc;
    },
    {} as Record<Language, number>,
  );

  const frameworks = Object.values(templates).reduce(
    (acc, { framework }) => {
      if (!framework) {
        return acc;
      }
      if (!acc[framework]) {
        acc[framework] = 0;
      }
      acc[framework]++;
      return acc;
    },
    {} as Record<Framework, number>,
  );

  const fields = form.watch();

  useEffect(() => {
    console.log("fields", fields);

    updateUrl(fields);
  }, [fields]);

  const filteredTemplates = useMemo(
    () =>
      Object.entries(templates).reduce(
        (acc, [id, template]) => {
          if (
            fields.frameworks.length > 0 &&
            (!template.framework || !fields.frameworks.includes(template.framework))
          ) {
            return acc;
          }
          if (fields.languages.length > 0 && !fields.languages.includes(template.language)) {
            return acc;
          }
          if (
            fields.search &&
            !template.title.toLowerCase().includes(fields.search.toLowerCase()) &&
            !template.description.toLowerCase().includes(fields.search.toLowerCase())
          ) {
            return acc;
          }
          acc[id] = template;
          return acc;
        },
        {} as typeof templates,
      ),
    [fields],
  );

  return (
    <>
      <PageIntro title="Find your Template">
        <p className="text-white/60 mt-10">
          Jumpstart your api development with our pre-built solutions.
        </p>
      </PageIntro>

      <Container className="pt-16 mt-24 text-white">
        <div className="flex flex-col lg:space-x-8 lg:flex-row mb-24">
          <div className="w-full lg:w-[232px]">
            <Form {...form}>
              <h2 className="mb-8 font-semibold blog-heading-gradient w-fit">Filter Templates</h2>
              <FormField
                control={form.control}
                name="search"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <SearchInput
                        placeholder="Search"
                        {...field}
                        className="rounded-lg border-[.75px] border-white/20 lg:w-[232px]"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <Separator className="mt-8 mb-8" orientation="horizontal" />

              <FormField
                control={form.control}
                name="languages"
                render={() => (
                  <FormItem className="mt-8 mb-4">
                    <Accordion type="single" collapsible>
                      <AccordionItem value="langAccordion">
                        <AccordionTrigger className="items-start text-left w-full">
                          <span className="w-6 h-6 rounded-md bg-white/10">
                            <CodeIcon />
                          </span>
                          <span className="text-left justify-start w-full pl-4">Languages</span>
                        </AccordionTrigger>

                        <AccordionContent>
                          <Separator className="mt-8 mb-8" orientation="horizontal" />
                          {Object.entries(languages).map(([language, occurences]) => (
                            <FormField
                              key={language}
                              control={form.control}
                              name="languages"
                              render={({ field }) => {
                                return (
                                  <FormItem
                                    key={language}
                                    className="flex flex-row items-center p-2 space-x-3 h-12 space-y-0 duration-150 rounded-md bg-[rgba(255,255,255,0.05)] group hover:bg-[rgba(255,255,255,0.15)] mb-2"
                                  >
                                    <FormControl>
                                      <Checkbox
                                        className="ml-2"
                                        checked={field.value?.includes(language)}
                                        onCheckedChange={(checked) => {
                                          return checked
                                            ? field.onChange([...field.value, language])
                                            : field.onChange(
                                                field.value?.filter(
                                                  (value: string) => value !== language,
                                                ),
                                              );
                                        }}
                                      />
                                    </FormControl>
                                    <FormLabel className="flex items-center justify-between w-full">
                                      <span className="text-sm font-normal">{language}</span>
                                      <span className="p-1 px-4 text-xs text-white/70 duration-150 bg-white/20 rounded-full group-hover:text-white/80">
                                        {occurences}
                                      </span>
                                    </FormLabel>
                                  </FormItem>
                                );
                              }}
                            />
                          ))}

                          <FormMessage />
                        </AccordionContent>
                      </AccordionItem>
                    </Accordion>
                  </FormItem>
                )}
              />
              <Separator className="mt-8 mb-8" orientation="horizontal" />
              <FormField
                control={form.control}
                name="frameworks"
                render={() => (
                  <FormItem className="mt-8 mb-4">
                    <Accordion type="single" collapsible>
                      <AccordionItem value="langAccordion">
                        <AccordionTrigger className="items-start text-left w-full">
                          <span className="w-6 h-6 rounded-md bg-white/10">
                            <FrameworkIcon />
                          </span>
                          <span className="text-left justify-start w-full pl-4">Framework</span>
                        </AccordionTrigger>

                        <AccordionContent>
                          <Separator className="mt-8 mb-8" orientation="horizontal" />
                          {Object.entries(frameworks).map(([framework, occurences]) => (
                            <FormField
                              key={framework}
                              control={form.control}
                              name="frameworks"
                              render={({ field }) => {
                                return (
                                  <FormItem
                                    key={framework}
                                    className="flex flex-row items-center h-12 p-2 space-x-3 space-y-0 duration-150 rounded-md bg-[rgba(255,255,255,0.05)] group hover:bg-[rgba(255,255,255,0.15)] mb-2"
                                  >
                                    <FormControl>
                                      <Checkbox
                                        checked={field.value?.includes(framework)}
                                        onCheckedChange={(checked) => {
                                          return checked
                                            ? field.onChange([...field.value, framework])
                                            : field.onChange(
                                                field.value?.filter(
                                                  (value: string) => value !== framework,
                                                ),
                                              );
                                        }}
                                      />
                                    </FormControl>
                                    <FormLabel className="flex items-center justify-between w-full">
                                      <span className="text-sm font-normal">{framework}</span>
                                      <span className="p-1 px-4 text-xs text-white/70 duration-150 bg-white/20 rounded-full group-hover:text-white/80">
                                        {occurences}
                                      </span>
                                    </FormLabel>
                                  </FormItem>
                                );
                              }}
                            />
                          ))}
                          <FormMessage />
                        </AccordionContent>
                      </AccordionItem>
                    </Accordion>
                  </FormItem>
                )}
              />
            </Form>
          </div>
          <div className="grid w-full grid-cols-1 gap-8 lg:w-3/4 lauto-rows-fr lg:grid-cols-3 md:grid-cols-2">
            {Object.entries(filteredTemplates).map(([id, template]) => (
              <Link
                key={id}
                href={`/templates/${id}`}
                className="flex flex-col items-start overflow-hidden duration-200 border min-h-96 border-white/10 shadow rounded-3xl hover:shadow-md hover:scale-[1.01]"
              >
                <div className="relative flex justify-center items-center h-2/5 w-full border-[.75px] border-white/10">
                  {template.image ? (
                    <img src={template.image} alt="" className="object-cover w-full h-full" />
                  ) : (
                    <VenetianMask className="w-16 h-16 text-white/60" />
                  )}
                </div>
                <div className="flex flex-col justify-start h-3/5 p-4">
                  <div>
                    <div className="flex flex-row  w-full justify-start gap-3">
                      {template.framework !== undefined ? (
                        <div className="py-0 px-2 rounded-sm bg-white/10 text-white/60 text-xs">
                          {template.framework?.toString()}
                        </div>
                      ) : null}
                      {template.language !== undefined ? (
                        <div className="py-0 px-3 rounded-sm bg-white/10 text-white/60 text-xs">
                          {template.language?.toString()}
                        </div>
                      ) : null}
                    </div>
                    <h3 className="mt-4 text-lg font-semibold leading-6 text-white group-hover:text-gray-600 line-clamp-2">
                      {template.title}
                    </h3>
                    <p className="mt-5 text-sm leading-6 text-white/60 line-clamp-2 ">
                      {template.description}
                    </p>
                  </div>
                  <div className="flex items-end justify-between h-full">
                    {/* No images currently in author */}
                    {/* <Avatar className="w-8 h-8 rounded-full" >
                      <AvatarImage src={template.authors} />
                    </Avatar> */}
                    <p className="text-sm leading-6 text-white ml-2">
                      by {template.authors.join(", ")}
                    </p>
                    <TemplatesRightArrow className="w-4 h-4 text-white/60 mr-2" />
                  </div>
                </div>
              </Link>
            ))}
          </div>
        </div>
      </Container>
      <CTA />
    </>
  );
}
