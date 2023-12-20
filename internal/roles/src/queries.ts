type RuleSet =
  | {
      $and: (RuleSet | string)[];
      $or?: never;
    }
  | {
      $and?: never;
      $or: (RuleSet | string)[];
    };

type Operation = (...args: (RuleSet | string)[]) => RuleSet;

const merge = (op: keyof RuleSet, ...args: (RuleSet | string)[]): RuleSet => {
  return args.reduce((acc: RuleSet, arg) => {
    if (!acc[op]) {
      acc[op] = [];
    }
    acc[op]!.push(arg);
    return acc;
  }, {} as RuleSet);
};

export const or: Operation = (...args) => merge("$or", ...args);
export const and: Operation = (...args) => merge("$and", ...args);

const ruleSet = and("abc", or("def", "ghi", and("jkl", "mno")));

console.log(JSON.stringify(ruleSet, null, 2));
