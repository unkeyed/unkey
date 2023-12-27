/**
 *  Here, the Result type is still a generic type that takes in a type T that extends the Actions
 *  type. It uses a mapped type to iterate over the keys of the T object and create a string literal
 *  union of all the possible combinations of resourceId:action strings. The [keyof T] at the end of
 *  the type definition means that the resulting type is a union of all the possible string literal
 *  unions created by the mapped type.
 *
 *  In the example, we define a new MyActions type that matches the Actions type from the original
 *  question, and then use the Result type to transform it into the desired MyResult type. The
 *  resulting type is "team::read" | "team::write", which matches the expected output.
 *
 *
 *  @example
 * type Resources = {
 *   team: 'read' | 'write';
 * };
 *
 * type MyResult = Flatten<Resources>; // type MyResult = "team::read" | "team::write"
 */

export type Flatten<T extends Record<string, string>, Delimiter extends string = "."> = {
  [K in keyof T]: `${K & string}${Delimiter}${T[K] & string}`;
}[keyof T];
