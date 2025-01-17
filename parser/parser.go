package parser

// TODO - don't bother with any of this rune business
// just parse strings and ignore most of the text
// whole thing honestly needs a rewrite

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type Powerfile struct {
	Name    string
	Options []Option
	Setups  map[string]Codeblock
	Tools   []Tool
}

type Tool struct {
	Name    string
	Options []Option
	Command Codeblock
}

type Codeblock struct {
	Text     string
	Language string
}

type Option struct {
	Name          string
	DefaultValue  string   // this will be an empty string if the option is required
	PossibleFlags []string // empty if a positional option
	IsBoolean     bool
	Position      int // -1 if not a positional option
}

type Header struct {
	Text  []string // slice of lines of text
	Order int      // number of octothorpes (#) preceding it
	Title string
}

// TODO - everything should have a "soft error"
// if there's an error, we report it, but keep going to try to get
// something useable
func ParsePowerfile(content string) (Powerfile, []error) {
	powerfile := Powerfile{}
	lines := WashText(content)
	headers, err := GetAllHeaders(lines)

	if err != nil {
		return powerfile, []error{err}
	}

	if len(headers) == 0 {
		return powerfile, []error{errors.New("no headers in markdown file")}
	}

	titleHeader := headers[0]

	if titleHeader.Order != 1 {
		return powerfile, []error{errors.New("powerfile does not start with a title")}
	}

	powerfile.Name = string(titleHeader.Title)

	i := 1
	errs := []error{}

	// find the base options
	for i < len(headers) {
		header := headers[i]
		title := string(header.Title)

		if strings.ToLower(title) == "options" && header.Order == 2 {
			options, err := ParseOptions(header.Text)
			powerfile.Options = options

			if err != nil {
				errs = append(errs, err)
			}
		}
		i += 1
	}

	// find the setup instructions
	i = 1
	mode := "looking"
	setupHeaders := []Header{}

	for i < len(headers) {
		header := headers[i]
		if mode == "looking" {
			title := string(header.Title)
			if strings.ToLower(title) == "setup" && header.Order == 2 {
				mode = "adding"
			}

		} else {
			if header.Order != 3 {
				break
			}
			setupHeaders = append(setupHeaders, header)
		}

		i += 1
	}

	setups, moreErrs := ParseSetup(setupHeaders)
	for _, err := range moreErrs {
		errs = append(errs, err)
	}
	powerfile.Setups = setups

	headerSplits := []struct {
		name       string
		subHeaders []Header
	}{}

	currentName := ""
	currentHeaders := []Header{}
	mode = "searching"

	i = 1

	for i < len(headers) {
		header := headers[i]

		if mode == "searching" {
			lowerName := strings.ToLower(header.Title)
			fmt.Println(lowerName)
			if header.Order == 2 && strings.HasPrefix(lowerName, "command: ") {
				currentName = header.Title[9:]
				mode = "adding"
			}

		} else if mode == "adding" {
			fmt.Println("Adding: ", header.Title)
			if header.Order != 3 {
				headerSplits = append(headerSplits, struct {
					name       string
					subHeaders []Header
				}{
					name:       currentName,
					subHeaders: currentHeaders,
				})

				mode = "searching"
				i -= 1
				currentHeaders = []Header{}
			} else {
				currentHeaders = append(currentHeaders, header)
			}
		}

		i += 1
	}

	if mode == "adding" {
		headerSplits = append(headerSplits, struct {
			name       string
			subHeaders []Header
		}{
			name:       currentName,
			subHeaders: currentHeaders,
		})
	}

	for _, headerSplit := range headerSplits {
		tool, err := ParseTool(headerSplit.name, headerSplit.subHeaders)

		if err != nil {
			errs = append(errs, err)
		}

		powerfile.Tools = append(powerfile.Tools, tool)
	}

	return powerfile, errs
}

func ParseTool(name string, headers []Header) (Tool, error) {
	tool := Tool{}
	tool.Name = name

	if len(headers) == 0 {
		return tool, errors.New("no headers to parse tool from")
	}

	for _, header := range headers {
		title := strings.ToLower(string(header.Title))

		if header.Order == 3 && title == "command" {
			codeblock, err := ParseCodeblock(header.Text)
			tool.Command = codeblock

			if err != nil {
				return tool, err
			}
		}

		if header.Order == 3 && title == "options" {
			options, err := ParseOptions(header.Text)
			tool.Options = options

			if err != nil {
				return tool, err
			}
		}
	}

	// TODO - what if we didn't see a codeblock or options?
	return tool, nil
}

func ParseSetup(headers []Header) (map[string]Codeblock, []error) {
	setup := map[string]Codeblock{}
	errs := []error{}

	for _, header := range headers {
		system := strings.ToLower(string(header.Title))
		codeblock, err := ParseCodeblock(header.Text)

		if err != nil {
			errs = append(errs, err)
		} else {
			setup[system] = codeblock
		}
	}

	return setup, errs
}

// we want to:
// - remove all \r
// - remove any empty lines for convinience
func WashText(content string) []string {
	text := []string{}

	content = strings.ReplaceAll(content, "\r", "")
	lines := strings.Split(content, "\n")
	inCodeblock := false

	for _, line := range lines {
		if inCodeblock {
			text = append(text, line)
		} else {
			isWhitespace := true

			for _, c := range line {
				if !(c == ' ' || c == '\t') {
					isWhitespace = false
					break
				}
			}

			if !isWhitespace {
				text = append(text, line)
			}
		}

		if strings.HasPrefix(line, "```") {
			inCodeblock = !inCodeblock
		}
	}

	return text
}

func GetCodeblockLines(lines []string) map[int]bool {
	codeblocks := map[int]bool{}
	inCodeblock := false

	for i, line := range lines {
		if strings.HasPrefix(line, "```") {
			inCodeblock = !inCodeblock
		} else if inCodeblock {
			codeblocks[i] = true
		} else {
			codeblocks[i] = false
		}
	}

	return codeblocks
}

func GetAllHeaders(lines []string) ([]Header, error) {
	headers := []Header{}

	header := Header{}
	currentText := []string{}
	codeblocks := GetCodeblockLines(lines)

	var i int = 0
	// skip until we have a header
	for i < len(lines) {
		hashes, title, err := ParseHeaderLine(lines[i])

		if err != nil {
			return nil, err
		}
		if hashes != 0 && codeblocks[i] == false {
			header.Order = hashes
			header.Title = title
			i += 1
			break
		}

		i += 1
	}

	for i < len(lines) {
		hashes, title, err := ParseHeaderLine(lines[i])

		if err != nil {
			return nil, err
		}

		if hashes != 0 && codeblocks[i] == false {
			header.Text = currentText
			headers = append(headers, header)

			header = Header{}
			header.Title = title
			header.Order = hashes

			currentText = []string{}
		} else {
			currentText = append(currentText, lines[i])
		}

		i += 1
	}

	header.Text = currentText
	headers = append(headers, header)

	return headers, nil
}

func ParseHeaderLine(line string) (int, string, error) {
	if len(line) < 1 {
		// this is reachable. It means that it's not a header
		return 0, "", nil
	}

	octothorpes := 0
	text := ""

	i := 0

	for line[i] == '#' && i < len(line) {
		octothorpes += 1

		i += 1
	}

	for (line[i] == ' ' || line[i] == '\t') && i < len(line) {
		i += 1
	}

	for j, c := range line {
		if i-1 < j {
			text += string(c)
		}
	}

	return octothorpes, text, nil
}

func ParseOptions(lines []string) ([]Option, error) {
	options := []Option{}

	for _, line := range lines {
		if len(line) == 0 {
			panic("Somehow got a line of length 0. Invalides a programmer assumption.")
		}

		if line[0] == '-' {
			option, err := ParseOptionLine(line)

			if err != nil {
				return options, errors.Join(errors.New("Error parsing options: "), err)
			}

			options = append(options, option)
		}
	}

	return options, nil
}

func ParseOptionLine(line string) (Option, error) {
	option := Option{}

	line = strings.TrimLeft(line, " \t\n")
	if len(line) == 0 {
		return option, errors.New("Length of line is 0")
	}

	if line[0] != '-' {
		return option, errors.New("no dash at the start of the line")
	}

	firstSplit := strings.Split(line, "=")

	if len(firstSplit) != 2 {
		return option, errors.New("first split does not have a length of 2")
	}

	// option is in three parts:
	// - `-o`, `--option` = `default` > "option"
	//   [^ nouns]   [^ hint ]   [^ namePart]
	//
	// we have to split it into those parts
	secondSplit := strings.Split(firstSplit[1], ">")

	if len(secondSplit) != 2 {
		return option, errors.New("No > to denote the name of the option")
	}

	nouns := firstSplit[0]
	hint := secondSplit[0]
	namePart := secondSplit[1]

	// now, just parse accordingly

	result, err := parseOptionHint(hint)
	if err != nil {
		return option, err
	}
	flags, err := parseOptionNouns(nouns)
	if err != nil {
		return option, err
	}
	name, err := parseOptionName(namePart)
	if err != nil {
		return option, err
	}

	option.Name = name
	option.PossibleFlags = flags
	option.DefaultValue = result.defaultValue
	option.Position = result.position
	option.IsBoolean = result.isBoolean

	if !isValidOptionName(option.Name) {
		return option, errors.New("Invalid option name: " + option.Name)
	}

	return option, nil
}

func parseOptionNouns(nouns string) ([]string, error) {
	flagStrings := []string{}
	flags := ExtractInsideString(nouns, '`')

	if len(flags) == 0 {
		return flagStrings, errors.New("No flags found")
	}

	for _, flag := range flags {
		flagStrings = append(flagStrings, string(flag))
	}

	return flagStrings, nil
}

// default value, isBoolean, or position
type hintResult = struct {
	defaultValue string
	isBoolean    bool
	position     int
}

func parseOptionHint(hint string) (hintResult, error) {
	hint = strings.Trim(hint, " \n\t")
	result := hintResult{
		isBoolean:    false,
		position:     -1,
		defaultValue: "",
	}

	if len(hint) == 0 {
		return result, errors.New("hint part was only whitespace")
	}

	// debug print
	// fmt.Println("Parsing hint:", string(hint))

	if len(hint) == 1 && hint[0] == '|' {
		result.isBoolean = true
		return result, nil
	}

	defaultExtraction := ExtractInsideString(hint, '`')

	if len(defaultExtraction) > 1 {
		return result, errors.New("found more than one default value")
	}

	if len(defaultExtraction) == 1 {
		result.defaultValue = string(defaultExtraction[0])
		return result, nil
	}

	num, err := strconv.Atoi(string(hint))

	if err != nil {
		return result, errors.Join(
			errors.New("error parsing positional option"), err)
	}

	result.position = num
	return result, nil
}

func parseOptionName(namePart string) (string, error) {
	extraction := ExtractInsideString(namePart, '"')

	if len(extraction) != 1 {
		return "", errors.New("None or many extracted names were found")
	}

	return string(extraction[0]), nil
}

func parseFlags(flagsText string) []string {
	inside := false
	current := ""
	flags := []string{}

	for _, c := range flagsText {
		if inside {
			if c == '`' {
				inside = false
				flags = append(flags, current)
				current = ""
			} else {
				current += string(c)
			}
		} else {
			if c == '`' {
				inside = true
			}
		}

	}

	return flags
}

func ParseCodeblock(lines []string) (Codeblock, error) {
	codeblock := Codeblock{
		Text:     "",
		Language: "",
	}

	var err error = nil

	inside := false
	for _, line := range lines {
		if inside {
			if strings.HasPrefix(line, "```") {
				break
			}
			codeblock.Text += string(line) + "\n"
		} else {
			if strings.HasPrefix(line, "```") {
				codeblock.Language = string(line[3:])
				inside = true
			}
		}
	}

	if inside == false {
		err = errors.New("no codeblock found")
	}

	return codeblock, err
}

func ExtractInsideString(text string, delimeter rune) []string {
	parts := []string{}
	current := ""
	inside := false

	for _, c := range text {
		if inside {
			if c == delimeter {
				inside = false
				parts = append(parts, current)
				current = ""
			} else {
				current += string(c)
			}
		} else {
			if c == delimeter {
				inside = true
			}
		}
	}

	return parts
}

// TODO - fix this
func isValidOptionName(name string) bool {
	validCharacters := []rune("abcdefghijklmnopqrstuvwxyz1234567890-_")
	for _, c := range name {
		if !slices.Contains(validCharacters, c) {
			return false
		}
	}

	return true
}
