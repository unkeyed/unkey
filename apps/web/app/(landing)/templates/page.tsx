"use client";
import { Container } from "@/components/landing/container";
import { PageIntro } from "@/components/landing/page-intro";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  TemplatesFormValues,
  getDefaulTemplatesFormValues,
  schema,
  updateUrl,
} from "@/lib/templates-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ExternalLink, VenetianMask } from "lucide-react";
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

  const languages = Object.values(templates).reduce((acc, { language }) => {
    if (!acc[language]) {
      acc[language] = 0;
    }
    acc[language]++;
    return acc;
  }, {} as Record<Language, number>);

  const frameworks = Object.values(templates).reduce((acc, { framework }) => {
    if (!framework) {
      return acc;
    }
    if (!acc[framework]) {
      acc[framework] = 0;
    }
    acc[framework]++;
    return acc;
  }, {} as Record<Framework, number>);

  const fields = form.watch();

  useEffect(() => {
    updateUrl(fields);
  }, [fields]);

  const filteredTemplates = useMemo(
    () =>
      Object.entries(templates).reduce((acc, [id, template]) => {
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
      }, {} as typeof templates),
    [fields],
  );

  return (
    <>
      <PageIntro title="Find your Template">
        <p>Jumpstart your api development with our pre-built solutions.</p>
      </PageIntro>

      <Container className="pt-16 mt-24 border-t">
        <div className="flex flex-col lg:space-x-8 lg:flex-row ">
          <div className="w-full lg:w-1/4">
            <Form {...form}>
              <h2 className="mb-8 font-semibold">Filter Templates</h2>

              <FormField
                control={form.control}
                name="search"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <Input placeholder="Search ..." {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="languages"
                render={() => (
                  <FormItem className="mt-8 mb-4">
                    <FormLabel className="text-base">Languages</FormLabel>
                    <FormDescription>
                      Select the programming languages you want to explore.
                    </FormDescription>
                    {Object.entries(languages).map(([language, occurences]) => (
                      <FormField
                        key={language}
                        control={form.control}
                        name="languages"
                        render={({ field }) => {
                          return (
                            <FormItem
                              key={language}
                              className="flex flex-row items-center p-2 space-x-3 space-y-0 duration-150 rounded bg-gray-50 group hover:bg-gray-100"
                            >
                              <FormControl>
                                <Checkbox
                                  checked={field.value?.includes(language)}
                                  onCheckedChange={(checked) => {
                                    console.log({ checked, field, language });
                                    return checked
                                      ? field.onChange([...field.value, language])
                                      : field.onChange(
                                          field.value?.filter((value) => value !== language),
                                        );
                                  }}
                                />
                              </FormControl>
                              <FormLabel className="flex items-center justify-between w-full">
                                <span className="text-sm font-normal">{language}</span>
                                <span className="p-1 text-xs text-gray-500 duration-150 bg-gray-100 rounded-full group-hover:text-gray-800">
                                  {occurences}
                                </span>
                              </FormLabel>
                            </FormItem>
                          );
                        }}
                      />
                    ))}
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="frameworks"
                render={() => (
                  <FormItem className="mt-8 mb-4">
                    <FormLabel className="text-base">Frameworks</FormLabel>
                    <FormDescription>Fancy a specific framework? Select it here.</FormDescription>
                    {Object.entries(frameworks).map(([framework, occurences]) => (
                      <FormField
                        key={framework}
                        control={form.control}
                        name="frameworks"
                        render={({ field }) => {
                          return (
                            <FormItem
                              key={framework}
                              className="flex flex-row items-center p-2 space-x-3 space-y-0 duration-150 rounded bg-gray-50 group hover:bg-gray-100"
                            >
                              <FormControl>
                                <Checkbox
                                  checked={field.value?.includes(framework)}
                                  onCheckedChange={(checked) => {
                                    console.log({ checked, field, framework });
                                    return checked
                                      ? field.onChange([...field.value, framework])
                                      : field.onChange(
                                          field.value?.filter((value) => value !== framework),
                                        );
                                  }}
                                />
                              </FormControl>
                              <FormLabel className="flex items-center justify-between w-full">
                                <span className="text-sm font-normal">{framework}</span>
                                <span className="p-1 text-xs text-gray-500 duration-150 bg-gray-100 rounded-full group-hover:text-gray-800">
                                  {occurences}
                                </span>
                              </FormLabel>
                            </FormItem>
                          );
                        }}
                      />
                    ))}
                    <FormMessage />
                  </FormItem>
                )}
              />
            </Form>
          </div>
          <div className="grid w-full grid-cols-1 gap-8 lg:w-3/4 lauto-rows-fr lg:grid-cols-3">
            {Object.entries(filteredTemplates).map(([id, template]) => (
              <Link
                key={id}
                href={`/templates/${id}`}
                className="flex flex-col items-start h-96 overflow-hidden duration-200 border border-gray-200 shadow rounded-xl hover:shadow-md hover:scale-[1.01]"
              >
                <div className="relative flex justify-center items-center h-full w-full aspect-[16/9] sm:aspect-[2/1] lg:aspect-[3/2]">
                  {template.image ? (
                    <img src={template.image} alt="" className="object-cover w-full h-full" />
                  ) : (
                    <VenetianMask className="w-16 h-16 text-gray-200" />
                  )}
                </div>
                <div className="flex flex-col justify-between h-full px-4 pb-4">
                  <div>
                    <h3 className="mt-3 text-lg font-semibold leading-6 text-gray-900 group-hover:text-gray-600 line-clamp-2">
                      {template.title}
                    </h3>
                    <p className="mt-5 text-sm leading-6 text-gray-500 line-clamp-2">
                      {template.description}
                    </p>
                  </div>
                  <div className="flex items-center justify-between mt-5">
                    <p className="text-sm leading-6 text-gray-500 ">
                      by {template.authors.join(", ")}
                    </p>
                    <ExternalLink className="w-4 h-4 text-gray-400" />
                  </div>
                </div>
              </Link>
            ))}
          </div>
        </div>
      </Container>
    </>
  );
}
