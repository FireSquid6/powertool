import { parseActionFile } from "../lib/action";
import { describe, it, expect } from "bun:test";

const example1 = `
--- 
steps: 
  - step: run
    description: Runs a cool tool
    kit: me/hello
    tool: say-hello
    answers:
      - idk
    args:
      foo: bar
`;

const example2 = `
---
whatever: 123
`;

const example3 = `
---
steps:
  - step: switch
    description: Does a thingy
    conditions:
      - comparison: =
        key: foo
        value: bar
        steps:
          - step: run
            description: Runs a cool tool
            kit: me/hello
            tool: say-hello
            answers:
              - idk
            args:
              foo: bar
`;

const example4 = `
---
scratch:
  foo: bar
steps:
  - step: run
    description: Runs a cool tool
    kit: me/hello
    tool: say-hello
    answers:
      - idk
    args:
      foo: bar
`;

const example5 = `
---
steps:
  - step: run
    kit: me/no-description
    tool: default
    answers:
      - idk
    args:
      this: step has no description
`;

const example6 = `
---
steps:
  - step: run
    kit: me/no-tool
    description: default
    answers:
      - idk
    args:
      this:
        - you shouldn't
        - put
        - lists
        - in args. This throws an error
`;

describe("parseActionFile", () => {
  it("parses a simple action", () => {
    expect(parseActionFile(example1)).toEqual({
      scratch: new Map<string, string>(),
      steps: [
        {
          description: "Runs a cool tool",
          kit: "me/hello",
          tool: "say-hello",
          answers: ["idk"],
          args: [
            {
              key: "foo",
              value: "bar",
            },
          ],
        },
      ],
    });
  });
  it("parses a switch statement", () => {
    const expected = {
      scratch: new Map<string, string>(),
      steps: [
        {
          description: "Does a thingy",
          conditions: [
            {
              key: "foo",
              value: "bar",
              comparison: "=",
              steps: [
                {
                  description: "Runs a cool tool",
                  kit: "me/hello",
                  tool: "say-hello",
                  answers: ["idk"],
                  args: [
                    {
                      key: "foo",
                      value: "bar",
                    },
                  ],
                },
              ],
            },
          ],
        },
      ],
    };

    const result = parseActionFile(example3);
    // @ts-ignore
    console.log(result.steps[0].conditions);
    expect(Bun.deepEquals(result, expected)).toBeTrue();
  });

  it("throws an error if there are no steps", () => {
    expect(() => parseActionFile(example2)).toThrow("action file has no steps");
  });

  // parses scratch
  // throws error when bad args or questions are passed
  // parses switch steps
  // parses steps without a description
});
