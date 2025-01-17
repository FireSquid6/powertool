package cli

import (
	"errors"
	"fmt"
	"github.com/pwrtool/powertool/parser"
	"strconv"
	"strings"
)

const STRING_FALSE = "FALSE"
const STRING_TRUE = "TRUE"

func ParseCommandArgs(args []string, options []parser.Option) (map[string]string, error) {
	argMap := map[string]string{}
	unsatisfiedOptions := []string{}

	for _, option := range options {
		if option.IsBoolean {
			argMap[option.Name] = STRING_FALSE
		} else if option.DefaultValue != "" {
			argMap[option.Name] = option.DefaultValue
		} else {
			// option is required!
			unsatisfiedOptions = append(unsatisfiedOptions, option.Name)
		}
	}

	var i = 0
	var positional = 1

	for i < len(args) {
		fmt.Printf("Starting loop at %d\n", i)
		if !(isFlag(args[i])) {
			// it's a positional arg
			fmt.Println("Found positional argument")
			option, err := getOptionWithPosition(positional, options)

			if err != nil {
				return argMap, err
			}

			argMap[option.Name] = args[i]

			positional += 1
		} else {
			option, err := getOptionWithArg(args[i], options)
			if err != nil {
				return argMap, err
			}

			if option.IsBoolean {
				argMap[option.Name] = STRING_TRUE
			} else {
				if i+1 >= len(args) {
					return argMap, errors.New("No value for flag " + args[i])
				}
				i += 1

				value := args[i]

				if isFlag(value) {
					return argMap, errors.New("Expected value, got flag " + value)
				}

				argMap[option.Name] = value
			}

		}
		i += 1
	}

  for _, unsatisfiedOption := range unsatisfiedOptions {
    _, ok := argMap[unsatisfiedOption]

    if !ok {
      return argMap, errors.New("Did not satisfy option " + unsatisfiedOption)
    }
  }

	return argMap, nil

}

func getOptionWithArg(arg string, options []parser.Option) (parser.Option, error) {
	for _, option := range options {
		for _, flag := range option.PossibleFlags {
			if flag == arg {
				return option, nil
			}
		}
	}

	return parser.Option{}, errors.New("No option with argument " + arg)
}

func getOptionWithPosition(position int, options []parser.Option) (parser.Option, error) {
	for _, option := range options {
		if option.Position == position {
			return option, nil
		}
	}

	return parser.Option{}, errors.New("no option for position " + strconv.Itoa(position) + " found")
}

func isFlag(s string) bool {
	return (strings.HasPrefix(s, "-") || strings.HasPrefix(s, "--"))
}
