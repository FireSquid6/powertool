import { ParsedRunstring, generateRunstring } from ".";
import { describe, test, expect } from "bun:test";

describe("generateRunstring", () => {
  test("should deal with no answers", () => {
    const runstring: ParsedRunstring = {
      tool: "test",
      from: "/some/sort/of/path",
      arguments: new Map([
        ["a", "coolarg"],
        ["c", "23"],
      ]),
      autoAnswer: false,
      answers: [],
    };
    expect(generateRunstring(runstring)).toEqual(
      "tool:test;from:/some/sort/of/path;args:[a:coolarg,c:23,];autoAnswer:f;answers:[]"
    );
  });
  test("should deal with no args", () => {
    const runstring: ParsedRunstring = {
      tool: "test",
      from: "/some/sort/of/path",
      arguments: new Map(),
      autoAnswer: false,
      answers: [],
    };
    expect(generateRunstring(runstring)).toEqual(
      "tool:test;from:/some/sort/of/path;args:[];autoAnswer:f;answers:[]"
    );
  });
  test("should deal with answers", () => {
    const runstring: ParsedRunstring = {
      tool: "test",
      from: "/some/path/",
      arguments: new Map([
        ["a", "coolarg"],
        ["c", "23"],
      ]),
      autoAnswer: true,
      answers: ["y", "dothethng", "nah"],
    };
    expect(generateRunstring(runstring)).toEqual(
      "tool:test;from:/some/path/;args:[a:coolarg,c:23,];autoAnswer:t;answers:[y,dothethng,nah,]"
    );
  });
});
