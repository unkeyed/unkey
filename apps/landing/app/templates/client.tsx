"use client";
import { Container } from "@/components/container";
import { CTA } from "@/components/cta";
import { ChangelogLight } from "@/components/svg/changelog";
import { CodeIcon, FrameworkIcon } from "@/components/svg/template-page";
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
import { MeteorLinesAngular } from "@/components/ui/meteorLines";
import { Separator } from "@/components/ui/separator";
import {
  type TemplatesFormValues,
  getDefaulTemplatesFormValues,
  schema,
  updateUrl,
} from "@/lib/templates-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ArrowRight, VenetianMask } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useMemo } from "react";
import { useForm } from "react-hook-form";
import { type Framework, type Language, templates } from "./data";

export function TemplatesClient() {
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
    <div className="flex flex-col mx-auto py-10 lg:py-0 text-white/60">
      <div>
        <div className="relative -z-100 max-w-[1000px] mx-auto">
          <ChangelogLight className="w-full" />
        </div>
        <div className="w-full h-full overflow-clip -z-20">
          <MeteorLinesAngular
            number={1}
            xPos={0}
            speed={10}
            delay={5}
            className="overflow-hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={0}
            speed={10}
            delay={0}
            className="overflow-hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={100}
            speed={10}
            delay={7}
            className="overflow-hidden sm:hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={100}
            speed={10}
            delay={2}
            className="overflow-hidden sm:hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={200}
            speed={10}
            delay={7}
            className="overflow-hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={200}
            speed={10}
            delay={2}
            className="overflow-hidden"
          />
          <MeteorLinesAngular
            number={1}
            xPos={400}
            speed={10}
            delay={5}
            className="overflow-hidden sm:hidden md:block"
          />
          <MeteorLinesAngular
            number={1}
            xPos={400}
            speed={10}
            delay={0}
            className="overflow-hidden sm:hidden md:block"
          />
        </div>
      </div>

      <PageIntro title="Find your template">
        <p className="mt-[26px] text-base about-founders-text-gradient ">
          Jumpstart your API development with our pre-built solutions.
        </p>
      </PageIntro>
      <Container className="mt-24 text-white">
        <div className="flex flex-col mb-24 lg:space-x-8 lg:flex-row">
          <div className="w-full lg:w-[232px]">
            <Form {...form}>
              <h2 className="w-full mb-4 font-semibold text-left blog-heading-gradient">
                Filter templates
              </h2>
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
              <Accordion type="multiple" defaultValue={["languages", "frameworks"]}>
                <FormField
                  control={form.control}
                  name="languages"
                  render={() => (
                    <FormItem className="mt-0 mb-4">
                      <Separator className="mb-4 " orientation="horizontal" />

                      <AccordionItem value="languages">
                        <AccordionTrigger className="items-start w-full text-left">
                          <span className="w-6 h-6 rounded-md bg-white/10">
                            <CodeIcon />
                          </span>
                          <span className="justify-start w-full pl-4 text-sm font-medium text-left">
                            Languages
                          </span>
                        </AccordionTrigger>

                        <AccordionContent>
                          <Separator className="my-4 " orientation="horizontal" />
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
                                      <span className="p-1 px-4 text-xs duration-150 rounded-full text-white/70 bg-white/20 group-hover:text-white/80">
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
                    </FormItem>
                  )}
                />
                <Separator className="mt-4 mb-4" orientation="horizontal" />
                <FormField
                  control={form.control}
                  name="frameworks"
                  render={() => (
                    <FormItem className="mt-4 mb-4">
                      <AccordionItem value="frameworks">
                        <AccordionTrigger className="items-start w-full text-left">
                          <span className="w-6 h-6 rounded-md bg-white/10">
                            <FrameworkIcon />
                          </span>
                          <span className="justify-start w-full pl-4 text-sm text-left">
                            Framework
                          </span>
                        </AccordionTrigger>

                        <AccordionContent>
                          <Separator className="mt-4 mb-4" orientation="horizontal" />
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
                                      <span className="p-1 px-4 text-xs duration-150 rounded-full text-white/70 bg-white/20 group-hover:text-white/80">
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
                    </FormItem>
                  )}
                />
              </Accordion>
            </Form>
          </div>
          <div className="grid w-full grid-cols-1 gap-8 xl:w-3/4 auto-rows-fr xl:grid-cols-3 md:grid-cols-2 grid-col-1">
            {Object.entries(filteredTemplates).map(([id, template]) => (
              <Link
                key={id}
                href={`/templates/${id}`}
                className="flex flex-col items-start justify-between overflow-hidden duration-200 border rounded-xl border-white/10 hover:border-white/20"
              >
                <div className="relative w-full">
                  {template.image ? (
                    <Image
                      src={template.image}
                      alt=""
                      width={800}
                      height={400}
                      className="aspect-[16/9] w-full  bg-gray-100 object-cover  sm:aspect-[2/1] "
                    />
                  ) : (
                    <div className="flex items-center justify-center w-full h-full">
                      <VenetianMask className="w-8 h-8 text-white/60" />
                    </div>
                  )}
                </div>
                <div className="flex flex-col justify-start w-full p-4 h-3/5">
                  <div>
                    <div className="flex flex-row justify-start w-full gap-3">
                      {template.framework !== undefined ? (
                        <div className="px-2 py-1 text-xs rounded-md bg-white/10 text-white/60">
                          {template.framework?.toString()}
                        </div>
                      ) : null}
                      {template.language !== undefined ? (
                        <div className="px-2 py-1 text-xs rounded-md bg-white/10 text-white/60">
                          {template.language?.toString()}
                        </div>
                      ) : null}
                    </div>
                    <h3 className="mt-6 text-lg font-semibold leading-6 text-white group-hover:text-gray-600 line-clamp-2">
                      {template.title}
                    </h3>
                    <p className="mt-5 text-sm leading-6 text-white/60 line-clamp-2 ">
                      {template.description}
                    </p>
                  </div>
                  <div className="flex flex-col justify-end w-full h-full">
                    {/* No images currently in author */}
                    {/* <Avatar className="w-8 h-8 rounded-full" >
                      <AvatarImage src={template.authors} />
                    </Avatar> */}
                    <div className="relative w-full pb-4">
                      <p className="absolute left-0 text-xs leading-6 text-left text-white">
                        {template.authors.join(", ")}
                      </p>
                      <div className="absolute right-0">
                        <ArrowRight className="flex w-4 h-4 mr-2 text-white/40 " />
                      </div>
                    </div>
                  </div>
                </div>
              </Link>
            ))}
          </div>
        </div>
      </Container>
      <CTA />
    </div>
  );
}
