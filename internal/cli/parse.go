package cli

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/wensboy/quick_arch/internal/config"
	errs "github.com/wensboy/quick_arch/internal/error"
)

type document struct {
	Command Command `json:"command"`
}

type Command struct {
	Name        string    `json:"name,omitempty"`
	Aliases     []string  `json:"aliases,omitempty"`
	Usage       string    `json:"usage,omitempty"`
	UsageText   string    `json:"usageText,omitempty"`
	ArgsUsage   string    `json:"argsUsage,omitempty"`
	Description string    `json:"description,omitempty"`
	Version     string    `json:"version,omitempty"`
	Category    string    `json:"category,omitempty"`
	Hidden      bool      `json:"hidden,omitempty"`
	HideHelp    bool      `json:"hideHelp,omitempty"`
	Flags       []Flag    `json:"flags,omitempty"`
	Commands    []Command `json:"commands,omitempty"`
}

type Flag struct {
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases,omitempty"`
	Usage       string   `json:"usage,omitempty"`
	Type        string   `json:"type"`
	Required    bool     `json:"required,omitempty"`
	Hidden      bool     `json:"hidden,omitempty"`
	DefaultText string   `json:"-"` // 仅用于 help 展示, 不参与取值
}

func ParseString(commandStr []byte) (*cli.Command, error) {
	var doc document
	if err := json.Unmarshal(commandStr, &doc); err != nil {
		return nil, errs.Wrap(ErrParseSource, err)
	}
	return buildCommand(&doc.Command), nil
}

func ParseFile(path string) (*cli.Command, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errs.Wrap(ErrReadSource, err).With("path", path)
	}
	return ParseString(data)
}

func buildCommand(cmd *Command) *cli.Command {
	if cmd == nil {
		return nil
	}

	out := &cli.Command{
		Name:        cmd.Name,
		Aliases:     cmd.Aliases,
		Usage:       cmd.Usage,
		UsageText:   cmd.UsageText,
		ArgsUsage:   cmd.ArgsUsage,
		Description: cmd.Description,
		Version:     cmd.Version,
		Category:    cmd.Category,
		Hidden:      cmd.Hidden,
		HideHelp:    cmd.HideHelp,
	}

	if len(cmd.Flags) > 0 {
		out.Flags = make([]cli.Flag, 0, len(cmd.Flags))
		for _, f := range cmd.Flags {
			out.Flags = append(out.Flags, BuildFlag(f))
		}
	}

	if len(cmd.Commands) > 0 {
		out.Commands = make([]*cli.Command, 0, len(cmd.Commands))
		for i := range cmd.Commands {
			out.Commands = append(out.Commands, buildCommand(&cmd.Commands[i]))
		}
	}

	return out
}

func BuildFlag(flag Flag) cli.Flag {
	switch normalizeFlagType(flag.Type) {
	case config.FlagTypeBool:
		return &cli.BoolFlag{
			Name:        flag.Name,
			Aliases:     flag.Aliases,
			Usage:       flag.Usage,
			Required:    flag.Required,
			Hidden:      flag.Hidden,
			DefaultText: flag.DefaultText,
		}
	case config.FlagTypeInt:
		return &cli.IntFlag{
			Name:        flag.Name,
			Aliases:     flag.Aliases,
			Usage:       flag.Usage,
			Required:    flag.Required,
			Hidden:      flag.Hidden,
			DefaultText: flag.DefaultText,
		}
	case config.FlagTypeFloat:
		return &cli.Float64Flag{
			Name:        flag.Name,
			Aliases:     flag.Aliases,
			Usage:       flag.Usage,
			Required:    flag.Required,
			Hidden:      flag.Hidden,
			DefaultText: flag.DefaultText,
		}
	default:
		return &cli.StringFlag{
			Name:        flag.Name,
			Aliases:     flag.Aliases,
			Usage:       flag.Usage,
			Required:    flag.Required,
			Hidden:      flag.Hidden,
			DefaultText: flag.DefaultText,
		}
	}
}

func normalizeFlagType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "int", "int8", "int16", "int32", "int64", "integer":
		return config.FlagTypeInt
	case "float", "float32", "float64", "number":
		return config.FlagTypeFloat
	case "bool", "boolean":
		return config.FlagTypeBool
	case "string", "str", "":
		return config.FlagTypeString
	default:
		return config.FlagTypeString
	}
}

func FlagToSource(cmd *cli.Command) map[string]any {
	source := make(map[string]any)
	if cmd == nil {
		return source
	}
	collectSetFlags(source, cmd, cmd.Name)
	return source
}

func collectSetFlags(out map[string]any, cmd *cli.Command, prefix string) {
	if cmd == nil {
		return
	}

	for _, fl := range cmd.Flags {
		if fl == nil || !fl.IsSet() {
			continue
		}
		names := fl.Names()
		if len(names) == 0 || names[0] == "" {
			continue
		}
		out[prefix+"."+names[0]] = fl.Get()
	}

	for _, sub := range cmd.Commands {
		if sub == nil || sub.Name == "" {
			continue
		}
		collectSetFlags(out, sub, prefix+"."+sub.Name)
	}
}
