/* eslint-disable @typescript-eslint/no-explicit-any */
/* eslint-disable @typescript-eslint/no-unused-vars */
import type {
  RouteConfig as RouteConfigBase,
  ZodContentObject,
  ZodMediaTypeObject,
  ZodRequestBody,
} from "@asteasolutions/zod-to-openapi";
import {
  OpenAPIRegistry,
  OpenApiGeneratorV3,
  OpenApiGeneratorV31,
  extendZodWithOpenApi,
} from "@asteasolutions/zod-to-openapi";
import { zValidator } from "@hono/zod-validator";
import { Hono } from "hono";
import type {
  Context,
  Env,
  Handler,
  Input,
  MiddlewareHandler,
  Schema,
  ToSchema,
  TypedResponse,
  ValidationTargets,
} from "hono";
import type { MergePath, MergeSchemaPath } from "hono/types";
import type { JSONParsed, RemoveBlankRecord } from "hono/utils/types";
import type {
  ClientErrorStatusCode,
  InfoStatusCode,
  RedirectStatusCode,
  ServerErrorStatusCode,
  StatusCode,
  SuccessStatusCode,
} from "hono/utils/http-status";
import { mergePath } from "hono/utils/url";
import type { ZodError, ZodSchema } from "zod";
import { ZodType, z } from "zod";

type MaybePromise<T> = Promise<T> | T;

export type RouteConfig = RouteConfigBase & {
  middleware?: MiddlewareHandler | MiddlewareHandler[];
};

type RequestTypes = {
  body?: ZodRequestBody;
  params?: ZodType;
  query?: ZodType;
  cookies?: ZodType;
  headers?: ZodType | ZodType[];
};

type IsJson<T> = T extends string
  ? T extends `application/${infer Start}json${infer _End}`
  ? Start extends "" | `${string}+` | `vnd.${string}+`
  ? "json"
  : never
  : never
  : never;

type IsForm<T> = T extends string
  ? T extends
  | `multipart/form-data${infer _Rest}`
  | `application/x-www-form-urlencoded${infer _Rest}`
  ? "form"
  : never
  : never;

type RequestPart<R extends RouteConfig, Part extends string> = Part extends keyof R["request"]
  ? R["request"][Part]
  : {};

type HasUndefined<T> = undefined extends T ? true : false;

type InputTypeBase<
  R extends RouteConfig,
  Part extends string,
  Type extends keyof ValidationTargets,
> = R["request"] extends RequestTypes
  ? RequestPart<R, Part> extends ZodType
  ? {
    in: {
      [K in Type]: HasUndefined<ValidationTargets[K]> extends true
      ? {
        [K2 in keyof z.input<RequestPart<R, Part>>]?: ValidationTargets[K][K2];
      }
      : {
        [K2 in keyof z.input<RequestPart<R, Part>>]: ValidationTargets[K][K2];
      };
    };
    out: { [K in Type]: z.output<RequestPart<R, Part>> };
  }
  : {}
  : {};

type InputTypeJson<R extends RouteConfig> = R["request"] extends RequestTypes
  ? R["request"]["body"] extends ZodRequestBody
  ? R["request"]["body"]["content"] extends ZodContentObject
  ? IsJson<keyof R["request"]["body"]["content"]> extends never
  ? {}
  : R["request"]["body"]["content"][keyof R["request"]["body"]["content"]] extends Record<
    "schema",
    ZodSchema<any>
  >
  ? {
    in: {
      json: z.input<
        R["request"]["body"]["content"][keyof R["request"]["body"]["content"]]["schema"]
      >;
    };
    out: {
      json: z.output<
        R["request"]["body"]["content"][keyof R["request"]["body"]["content"]]["schema"]
      >;
    };
  }
  : {}
  : {}
  : {}
  : {};

type InputTypeForm<R extends RouteConfig> = R["request"] extends RequestTypes
  ? R["request"]["body"] extends ZodRequestBody
  ? R["request"]["body"]["content"] extends ZodContentObject
  ? IsForm<keyof R["request"]["body"]["content"]> extends never
  ? {}
  : R["request"]["body"]["content"][keyof R["request"]["body"]["content"]] extends Record<
    "schema",
    ZodSchema<any>
  >
  ? {
    in: {
      form: z.input<
        R["request"]["body"]["content"][keyof R["request"]["body"]["content"]]["schema"]
      >;
    };
    out: {
      form: z.output<
        R["request"]["body"]["content"][keyof R["request"]["body"]["content"]]["schema"]
      >;
    };
  }
  : {}
  : {}
  : {}
  : {};

type InputTypeParam<R extends RouteConfig> = InputTypeBase<R, "params", "param">;
type InputTypeQuery<R extends RouteConfig> = InputTypeBase<R, "query", "query">;
type InputTypeHeader<R extends RouteConfig> = InputTypeBase<R, "headers", "header">;
type InputTypeCookie<R extends RouteConfig> = InputTypeBase<R, "cookies", "cookie">;

type ExtractContent<T> = T extends {
  [K in keyof T]: infer A;
}
  ? A extends Record<"schema", ZodSchema>
  ? z.infer<A["schema"]>
  : never
  : never;

type StatusCodeRangeDefinitions = {
  "1XX": InfoStatusCode;
  "2XX": SuccessStatusCode;
  "3XX": RedirectStatusCode;
  "4XX": ClientErrorStatusCode;
  "5XX": ServerErrorStatusCode;
};
type RouteConfigStatusCode = keyof StatusCodeRangeDefinitions | StatusCode;
type ExtractStatusCode<T extends RouteConfigStatusCode> = T extends keyof StatusCodeRangeDefinitions
  ? StatusCodeRangeDefinitions[T]
  : T;
export type RouteConfigToTypedResponse<R extends RouteConfig> = {
  [Status in keyof R["responses"] & RouteConfigStatusCode]: IsJson<
    keyof R["responses"][Status]["content"]
  > extends never
  ? TypedResponse<{}, ExtractStatusCode<Status>, string>
  : TypedResponse<
    JSONParsed<ExtractContent<R["responses"][Status]["content"]>>,
    ExtractStatusCode<Status>,
    "json" | "text"
  >;
}[keyof R["responses"] & RouteConfigStatusCode];

export type Hook<T, E extends Env, P extends string, R> = (
  result: { target: keyof ValidationTargets } & (
    | {
      success: true;
      data: T;
    }
    | {
      success: false;
      error: ZodError;
    }
  ),
  c: Context<E, P>,
) => R;

type ConvertPathType<T extends string> = T extends `${infer Start}/{${infer Param}}${infer Rest}`
  ? `${Start}/:${Param}${ConvertPathType<Rest>}`
  : T;

export type OpenAPIHonoOptions<E extends Env> = {
  defaultHook?: Hook<any, E, any, any>;
};
type HonoInit<E extends Env> = ConstructorParameters<typeof Hono>[0] & OpenAPIHonoOptions<E>;

export type RouteHandler<
  R extends RouteConfig,
  E extends Env = Env,
  I extends Input = InputTypeParam<R> &
  InputTypeQuery<R> &
  InputTypeHeader<R> &
  InputTypeCookie<R> &
  InputTypeForm<R> &
  InputTypeJson<R>,
  P extends string = ConvertPathType<R["path"]>,
> = Handler<
  E,
  P,
  I,
  // If response type is defined, only TypedResponse is allowed.
  R extends {
    responses: {
      [statusCode: number]: {
        content: {
          [mediaType: string]: ZodMediaTypeObject;
        };
      };
    };
  }
  ? MaybePromise<RouteConfigToTypedResponse<R>>
  : MaybePromise<RouteConfigToTypedResponse<R>> | MaybePromise<Response>
>;

export type RouteHook<
  R extends RouteConfig,
  E extends Env = Env,
  I extends Input = InputTypeParam<R> &
  InputTypeQuery<R> &
  InputTypeHeader<R> &
  InputTypeCookie<R> &
  InputTypeForm<R> &
  InputTypeJson<R>,
  P extends string = ConvertPathType<R["path"]>,
> = Hook<
  I,
  E,
  P,
  RouteConfigToTypedResponse<R> | Response | Promise<Response> | void | Promise<void>
>;

type OpenAPIObjectConfig = Parameters<
  InstanceType<typeof OpenApiGeneratorV3>["generateDocument"]
>[0];

export type OpenAPIObjectConfigure<E extends Env, P extends string> =
  | OpenAPIObjectConfig
  | ((context: Context<E, P>) => OpenAPIObjectConfig);

export class OpenAPIHono<
  E extends Env = Env,
  S extends Schema = {},
  BasePath extends string = "/",
> extends Hono<E, S, BasePath> {
  openAPIRegistry: OpenAPIRegistry;
  defaultHook?: OpenAPIHonoOptions<E>["defaultHook"];

  constructor(init?: HonoInit<E>) {
    super(init);
    this.openAPIRegistry = new OpenAPIRegistry();
    this.defaultHook = init?.defaultHook;
  }

  /**
   *
   * @param {RouteConfig} route - The route definition which you create with `createRoute()`.
   * @param {Handler} handler - The handler. If you want to return a JSON object, you should specify the status code with `c.json()`.
   * @param {Hook} hook - Optional. The hook method defines what it should do after validation.
   * @example
   * app.openapi(
   *   route,
   *   (c) => {
   *     // ...
   *     return c.json(
   *       {
   *         age: 20,
   *         name: 'Young man',
   *       },
   *       200 // You should specify the status code even if it's 200.
   *     )
   *   },
   *  (result, c) => {
   *    if (!result.success) {
   *      return c.json(
   *        {
   *          code: 400,
   *          message: 'Custom Message',
   *        },
   *        400
   *      )
   *    }
   *  }
   *)
   */
  openapi = <
    R extends RouteConfig,
    I extends Input = InputTypeParam<R> &
    InputTypeQuery<R> &
    InputTypeHeader<R> &
    InputTypeCookie<R> &
    InputTypeForm<R> &
    InputTypeJson<R>,
    P extends string = ConvertPathType<R["path"]>,
  >(
    { middleware: routeMiddleware, ...route }: R,
    handler: Handler<
      E,
      P,
      I,
      // If response type is defined, only TypedResponse is allowed.
      R extends {
        responses: {
          [statusCode: number]: {
            content: {
              [mediaType: string]: ZodMediaTypeObject;
            };
          };
        };
      }
      ? MaybePromise<RouteConfigToTypedResponse<R>>
      : MaybePromise<RouteConfigToTypedResponse<R>> | MaybePromise<Response>
    >,
    hook:
      | Hook<
        I,
        E,
        P,
        R extends {
          responses: {
            [statusCode: number]: {
              content: {
                [mediaType: string]: ZodMediaTypeObject;
              };
            };
          };
        }
        ? MaybePromise<RouteConfigToTypedResponse<R>> | undefined
        : MaybePromise<RouteConfigToTypedResponse<R>> | MaybePromise<Response> | undefined
      >
      | undefined = this.defaultHook,
  ): OpenAPIHono<
    E,
    S & ToSchema<R["method"], MergePath<BasePath, P>, I, RouteConfigToTypedResponse<R>>,
    BasePath
  > => {
    this.openAPIRegistry.registerPath(route);

    const validators: MiddlewareHandler[] = [];

    if (route.request?.query) {
      const validator = zValidator("query", route.request.query as any, hook as any);
      validators.push(validator as any);
    }

    if (route.request?.params) {
      const validator = zValidator("param", route.request.params as any, hook as any);
      validators.push(validator as any);
    }

    if (route.request?.headers) {
      const validator = zValidator("header", route.request.headers as any, hook as any);
      validators.push(validator as any);
    }

    if (route.request?.cookies) {
      const validator = zValidator("cookie", route.request.cookies as any, hook as any);
      validators.push(validator as any);
    }

    const bodyContent = route.request?.body?.content;

    if (bodyContent) {
      for (const mediaType of Object.keys(bodyContent)) {
        if (!bodyContent[mediaType]) {
          continue;
        }
        const schema = (bodyContent[mediaType] as ZodMediaTypeObject)["schema"];
        if (!(schema instanceof ZodType)) {
          continue;
        }
        if (isJSONContentType(mediaType)) {
          const validator = zValidator("json", schema, hook as any);
          if (route.request?.body?.required) {
            validators.push(validator);
          } else {
            const mw: MiddlewareHandler = async (c, next) => {
              if (c.req.header("content-type")) {
                if (isJSONContentType(c.req.header("content-type")!)) {
                  return await validator(c, next);
                }
              }
              c.req.addValidatedData("json", {});
              await next();
            };
            validators.push(mw);
          }
        }
        if (isFormContentType(mediaType)) {
          const validator = zValidator("form", schema, hook as any);
          if (route.request?.body?.required) {
            validators.push(validator);
          } else {
            const mw: MiddlewareHandler = async (c, next) => {
              if (c.req.header("content-type")) {
                if (isFormContentType(c.req.header("content-type")!)) {
                  return await validator(c, next);
                }
              }
              c.req.addValidatedData("form", {});
              await next();
            };
            validators.push(mw);
          }
        }
      }
    }

    const middleware = routeMiddleware
      ? Array.isArray(routeMiddleware)
        ? routeMiddleware
        : [routeMiddleware]
      : [];

    this.on(
      [route.method],
      route.path.replaceAll(/\/{(.+?)}/g, "/:$1"),
      ...middleware,
      ...validators,
      handler,
    );
    return this;
  };

  getOpenAPIDocument = (
    config: OpenAPIObjectConfig,
  ): ReturnType<typeof generator.generateDocument> => {
    const generator = new OpenApiGeneratorV3(this.openAPIRegistry.definitions);
    const document = generator.generateDocument(config);
    // @ts-expect-error the _basePath is a private property
    return this._basePath ? addBasePathToDocument(document, this._basePath) : document;
  };

  getOpenAPI31Document = (
    config: OpenAPIObjectConfig,
  ): ReturnType<typeof generator.generateDocument> => {
    const generator = new OpenApiGeneratorV31(this.openAPIRegistry.definitions);
    const document = generator.generateDocument(config);
    // @ts-expect-error the _basePath is a private property
    return this._basePath ? addBasePathToDocument(document, this._basePath) : document;
  };

  doc = <P extends string>(
    path: P,
    configure: OpenAPIObjectConfigure<E, P>,
  ): OpenAPIHono<E, S & ToSchema<"get", P, {}, {}>, BasePath> => {
    return this.get(path, (c) => {
      const config = typeof configure === "function" ? configure(c) : configure;
      try {
        const document = this.getOpenAPIDocument(config);
        return c.json(document);
      } catch (e: any) {
        return c.json(e, 500);
      }
    }) as any;
  };

  doc31 = <P extends string>(
    path: P,
    configure: OpenAPIObjectConfigure<E, P>,
  ): OpenAPIHono<E, S & ToSchema<"get", P, {}, {}>, BasePath> => {
    return this.get(path, (c) => {
      const config = typeof configure === "function" ? configure(c) : configure;
      try {
        const document = this.getOpenAPI31Document(config);
        return c.json(document);
      } catch (e: any) {
        return c.json(e, 500);
      }
    }) as any;
  };

  route<
    SubPath extends string,
    SubEnv extends Env,
    SubSchema extends Schema,
    SubBasePath extends string,
  >(
    path: SubPath,
    app: Hono<SubEnv, SubSchema, SubBasePath>,
  ): OpenAPIHono<E, MergeSchemaPath<SubSchema, MergePath<BasePath, SubPath>> & S, BasePath>;
  route<SubPath extends string>(path: SubPath): Hono<E, RemoveBlankRecord<S>, BasePath>;
  route<
    SubPath extends string,
    SubEnv extends Env,
    SubSchema extends Schema,
    SubBasePath extends string,
  >(
    path: SubPath,
    app?: Hono<SubEnv, SubSchema, SubBasePath>,
  ): OpenAPIHono<E, MergeSchemaPath<SubSchema, MergePath<BasePath, SubPath>> & S, BasePath> {
    const pathForOpenAPI = path.replaceAll(/:([^\/]+)/g, "{$1}");
    super.route(path, app as any);

    if (!(app instanceof OpenAPIHono)) {
      return this as any;
    }

    app.openAPIRegistry.definitions.forEach((def) => {
      switch (def.type) {
        case "component":
          return this.openAPIRegistry.registerComponent(def.componentType, def.name, def.component);

        case "route":
          return this.openAPIRegistry.registerPath({
            ...def.route,
            path: mergePath(pathForOpenAPI, def.route.path),
          });

        case "webhook":
          return this.openAPIRegistry.registerWebhook({
            ...def.webhook,
            path: mergePath(pathForOpenAPI, def.webhook.path),
          });

        case "schema":
          return this.openAPIRegistry.register(def.schema._def.openapi._internal.refId, def.schema);

        case "parameter":
          return this.openAPIRegistry.registerParameter(
            def.schema._def.openapi._internal.refId,
            def.schema,
          );

        default: {
          const errorIfNotExhaustive: never = def;
          throw new Error(`Unknown registry type: ${errorIfNotExhaustive}`);
        }
      }
    });

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return this as any;
  }

  basePath<SubPath extends string>(path: SubPath): OpenAPIHono<E, S, MergePath<BasePath, SubPath>> {
    return new OpenAPIHono({ ...(super.basePath(path) as any), defaultHook: this.defaultHook });
  }
}

type RoutingPath<P extends string> = P extends `${infer Head}/{${infer Param}}${infer Tail}`
  ? `${Head}/:${Param}${RoutingPath<Tail>}`
  : P;

export const createRoute = <P extends string, R extends Omit<RouteConfig, "path"> & { path: P }>(
  routeConfig: R,
) => {
  const route = {
    ...routeConfig,
    getRoutingPath(): RoutingPath<R["path"]> {
      return routeConfig.path.replaceAll(/\/{(.+?)}/g, "/:$1") as RoutingPath<P>;
    },
  };
  return Object.defineProperty(route, "getRoutingPath", { enumerable: false });
};

extendZodWithOpenApi(z);
export { extendZodWithOpenApi, z };

function addBasePathToDocument(document: Record<string, any>, basePath: string) {
  const updatedPaths: Record<string, any> = {};

  Object.keys(document.paths).forEach((path) => {
    updatedPaths[mergePath(basePath, path)] = document.paths[path];
  });

  return {
    ...document,
    paths: updatedPaths,
  };
}

function isJSONContentType(contentType: string) {
  return /^application\/([a-z-\.]+\+)?json/.test(contentType);
}

function isFormContentType(contentType: string) {
  return (
    contentType.startsWith("multipart/form-data") ||
    contentType.startsWith("application/x-www-form-urlencoded")
  );
}
