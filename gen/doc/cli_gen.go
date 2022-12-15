package gen

import (
	"bytes"
	"fmt"
	"github.com/urfave/cli/v2"
	"io"
	"sort"
	"strings"
	"text/template"
)

var MarkdownDocTemplate = `{{if gt .SectionNum 0}}% {{ .App.Name }} {{ .SectionNum }}

{{end}}# NAME

{{ .App.Name }}{{ if .App.Usage }} - {{ .App.Usage }}{{ end }}

# SYNOPSIS

{{ .App.Name }}
{{ if .SynopsisArgs }}
` + "```" + `
{{ range $v := .SynopsisArgs }}{{ $v }}{{ end }}` + "```" + `
{{ end }}{{ if .App.Description }}
# DESCRIPTION

{{ .App.Description }}
{{ end }}
**Usage**:

` + "```" + `{{ if .App.UsageText }}
{{ .App.UsageText }}
{{ else }}
{{ .App.Name }} [GLOBAL OPTIONS] command [COMMAND OPTIONS] [ARGUMENTS...]
{{ end }}` + "```" + `
{{ if .GlobalArgs }}
# GLOBAL OPTIONS
` + "```" +
	`{{ range $v := .GlobalArgs }}
{{ $v }}{{ end }}{{ end }}` + "```" +
	`{{ if .Commands }}
# COMMANDS
{{ range $v := .Commands }}
{{ $v }}{{ end }}{{ end }}`

// ToMarkdown creates a markdown string for the `*App`
// The function errors if either parsing or writing of the string fails.
func ToMarkdown(app *cli.App) (string, error) {
	var w bytes.Buffer
	if err := writeDocTemplate(app, &w, 0); err != nil {
		return "", err
	}
	return w.String(), nil
}

type cliTemplate struct {
	App          *cli.App
	SectionNum   int
	Commands     []string
	GlobalArgs   []string
	SynopsisArgs []string
}

func writeDocTemplate(a *cli.App, w io.Writer, sectionNum int) error {
	const name = "cli"
	t, err := template.New(name).Parse(MarkdownDocTemplate)
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(w, name, &cliTemplate{
		App:          a,
		SectionNum:   sectionNum,
		Commands:     prepareCommands(a.Commands, 0),
		GlobalArgs:   prepareArgsWithValues(a.VisibleFlags()),
		SynopsisArgs: prepareArgsSynopsis(a.VisibleFlags()),
	})
}

func prepareCommands(commands []*cli.Command, level int) []string {
	var coms []string
	for _, command := range commands {
		if command.Hidden {
			continue
		}

		usageText := prepareUsageText(command)

		usage := prepareUsage(command, usageText)

		prepared := fmt.Sprintf("%s %s\n\n%s%s",
			strings.Repeat("#", level+2),
			strings.Join(command.Names(), ", "),
			usage,
			usageText,
		)

		flags := prepareArgsWithValues(command.VisibleFlags())
		if len(flags) > 0 {
			prepared += fmt.Sprintf("\n**Options**")
			prepared += fmt.Sprintf("\n```")
			prepared += fmt.Sprintf("\n%s", strings.Join(flags, ""))
			prepared += fmt.Sprintf("```")
		}

		coms = append(coms, prepared)

		// recursively iterate subcommands
		if len(command.Subcommands) > 0 {
			coms = append(
				coms,
				prepareCommands(command.Subcommands, level+1)...,
			)
		}
	}

	return coms
}

func prepareArgsWithValues(flags []cli.Flag) []string {
	return prepareFlags(flags, ", ", "", "", `""`, true)
}

func prepareArgsSynopsis(flags []cli.Flag) []string {
	return prepareFlags(flags, "|", "[", "]", "[value]", false)
}

func prepareFlags(
	flags []cli.Flag,
	sep, opener, closer, value string,
	addDetails bool,
) []string {
	args := []string{}
	for _, f := range flags {
		flag, ok := f.(cli.DocGenerationFlag)
		if !ok {
			continue
		}
		modifiedArg := opener

		for _, s := range flag.Names() {
			trimmed := strings.TrimSpace(s)
			if len(modifiedArg) > len(opener) {
				modifiedArg += sep
			}
			if len(trimmed) > 1 {
				modifiedArg += fmt.Sprintf("--%s", trimmed)
			} else {
				modifiedArg += fmt.Sprintf("-%s", trimmed)
			}
		}
		modifiedArg += closer
		//if flag.TakesValue() {
		//	modifiedArg += fmt.Sprintf("=%s", value)
		//}

		if addDetails {
			modifiedArg = fmt.Sprintf("%-20s%s", modifiedArg, flagDetails(flag))
		}

		args = append(args, modifiedArg+"\n")

	}
	sort.Strings(args)
	return args
}

// flagDetails returns a string containing the flags metadata
func flagDetails(flag cli.DocGenerationFlag) string {
	description := flag.GetUsage()
	value := flag.GetValue()
	if value != "" {
		description += " (default: " + value + ")"
	}
	//return ": " + description
	return description
}

func prepareUsageText(command *cli.Command) string {
	if command.UsageText == "" {
		return ""
	}

	// Remove leading and trailing newlines
	preparedUsageText := strings.Trim(command.UsageText, "\n")

	var usageText string
	if strings.Contains(preparedUsageText, "\n") {
		// Format multi-line string as a code block using the 4 space schema to allow for embedded markdown such
		// that it will not break the continuous code block.
		for _, ln := range strings.Split(preparedUsageText, "\n") {
			usageText += fmt.Sprintf("    %s\n", ln)
		}
	} else {
		// Style a single line as a note
		usageText = fmt.Sprintf(">%s\n", preparedUsageText)
	}

	return usageText
}

func prepareUsage(command *cli.Command, usageText string) string {
	if command.Usage == "" {
		return ""
	}

	usage := command.Usage + "\n"
	// Add a newline to the Usage IFF there is a UsageText
	if usageText != "" {
		usage += "\n"
	}

	return usage
}
