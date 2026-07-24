import type { paths } from "./generated/schema.js";
import type { HttpMethod } from "./types.js";

export type ApiPaths = paths;
export type ApiPath = keyof ApiPaths;

export type ApiMethodForPath<Path extends ApiPath> = Extract<
  {
    [Method in HttpMethod]: ApiPaths[Path][Method] extends never
      ? never
      : Method;
  }[HttpMethod],
  HttpMethod
>;

export type ApiOperation<
  Path extends ApiPath,
  Method extends ApiMethodForPath<Path>,
> = ApiPaths[Path][Method];
