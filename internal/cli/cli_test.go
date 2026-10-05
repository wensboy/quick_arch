package cli

// internal/cli 全部单元测试记录在本文件.

import (
	"errors"
	"reflect"
	"testing"

	"github.com/urfave/cli/v3"
)

const exampleCommandFile = "../../data/conf/command.json"

func parseExample(t *testing.T) *cli.Command {
	t.Helper()

	root, err := ParseFile(exampleCommandFile)
	if err != nil {
		t.Fatalf("ParseFile(%q) error: %v", exampleCommandFile, err)
	}
	if root == nil {
		t.Fatal("ParseFile returned nil command")
	}
	return root
}

func setFlag(t *testing.T, cmd *cli.Command, name, value string) {
	t.Helper()

	for _, fl := range cmd.Flags {
		names := fl.Names()
		if len(names) > 0 && names[0] == name {
			if err := fl.Set(name, value); err != nil {
				t.Fatalf("set flag %q: %v", name, err)
			}
			return
		}
	}
	t.Fatalf("flag %q not found in command %q", name, cmd.Name)
}

func flagNames(cmd *cli.Command) []string {
	names := make([]string, 0, len(cmd.Flags))
	for _, fl := range cmd.Flags {
		if n := fl.Names(); len(n) > 0 {
			names = append(names, n[0])
		}
	}
	return names
}

func TestParseFile_BuildCommandTree(t *testing.T) {
	root := parseExample(t)

	if root.Usage == "" || root.Description == "" {
		t.Fatalf("root usage/description not mapped: %q / %q", root.Usage, root.Description)
	}

	if got := flagNames(root); !reflect.DeepEqual(got, []string{"config", "verbose"}) {
		t.Fatalf("root flag names = %v, want [config verbose]", got)
	}
	if _, ok := root.Flags[0].(*cli.StringFlag); !ok {
		t.Fatalf("root flag[0] type = %T, want *cli.StringFlag", root.Flags[0])
	}
	if _, ok := root.Flags[1].(*cli.BoolFlag); !ok {
		t.Fatalf("root flag[1] type = %T, want *cli.BoolFlag", root.Flags[1])
	}
	if got := root.Flags[0].Names(); !reflect.DeepEqual(got, []string{"config", "c"}) {
		t.Fatalf("config names = %v, want [config c]", got)
	}

	subNames := make([]string, 0, len(root.Commands))
	for _, c := range root.Commands {
		subNames = append(subNames, c.Name)
	}
	if !reflect.DeepEqual(subNames, []string{"version", "serve", "generate"}) {
		t.Fatalf("subcommands = %v, want [version serve generate]", subNames)
	}

	version := root.Command("version")
	if version == nil {
		t.Fatal("version command not found")
	}
	if _, ok := version.Flags[0].(*cli.BoolFlag); !ok {
		t.Fatalf("version flag type = %T, want *cli.BoolFlag", version.Flags[0])
	}

	serve := root.Command("srv")
	if serve == nil {
		t.Fatal("serve command not found by alias")
	}
	if !reflect.DeepEqual(serve.Aliases, []string{"srv"}) {
		t.Fatalf("serve aliases = %v", serve.Aliases)
	}
	if got := flagNames(serve); !reflect.DeepEqual(got, []string{"port", "rate"}) {
		t.Fatalf("serve flag names = %v", got)
	}
	if _, ok := serve.Flags[0].(*cli.IntFlag); !ok {
		t.Fatalf("serve flag[0] type = %T, want *cli.IntFlag", serve.Flags[0])
	}
	if _, ok := serve.Flags[1].(*cli.Float64Flag); !ok {
		t.Fatalf("serve flag[1] type = %T, want *cli.Float64Flag", serve.Flags[1])
	}
	if len(serve.Commands) != 1 || serve.Commands[0].Name != "rest" {
		t.Fatalf("serve subcommands = %v, want [rest]", serve.Commands)
	}
}

func TestBuildFlag_Normalize(t *testing.T) {
	cases := []struct {
		typ  string
		want cli.Flag
	}{
		{"string", &cli.StringFlag{}},
		{"str", &cli.StringFlag{}},
		{"", &cli.StringFlag{}},
		{"unknown", &cli.StringFlag{}},
		{"int", &cli.IntFlag{}},
		{"integer", &cli.IntFlag{}},
		{"int64", &cli.IntFlag{}},
		{"float", &cli.Float64Flag{}},
		{"number", &cli.Float64Flag{}},
		{"BOOL", &cli.BoolFlag{}},
		{"boolean", &cli.BoolFlag{}},
	}

	for _, tc := range cases {
		got := BuildFlag(Flag{Name: "x", Type: tc.typ})
		if reflect.TypeOf(got) != reflect.TypeOf(tc.want) {
			t.Errorf("BuildFlag(type=%q) = %T, want %T", tc.typ, got, tc.want)
		}
	}
}

func TestBuildFlag_FieldMapping(t *testing.T) {
	var fl cli.Flag = BuildFlag(Flag{
		Name:     "port",
		Aliases:  []string{"p"},
		Usage:    "监听端口",
		Type:     "int",
		Required: true,
		Hidden:   true,
	})
	got, ok := fl.(*cli.IntFlag)
	if !ok {
		t.Fatalf("BuildFlag returned %T, want *cli.IntFlag", fl)
	}
	if got.Name != "port" || !reflect.DeepEqual(got.Aliases, []string{"p"}) ||
		got.Usage != "监听端口" || !got.Required || !got.Hidden {
		t.Fatalf("flag fields not mapped: %+v", got)
	}
}

func TestFlagToSource_OnlySet(t *testing.T) {
	root := parseExample(t)
	root.Name = "quick_arch"

	setFlag(t, root, "config", "/etc/quick_arch.json")
	setFlag(t, root, "verbose", "true")
	setFlag(t, root.Command("serve"), "port", "8080")
	setFlag(t, root.Command("serve"), "rate", "1.5")

	got := FlagToSource(root)
	want := map[string]any{
		"quick_arch.config":     "/etc/quick_arch.json",
		"quick_arch.verbose":    true,
		"quick_arch.serve.port": 8080,
		"quick_arch.serve.rate": 1.5,
	}

	if len(got) != len(want) {
		t.Fatalf("source size = %d (%v), want %d", len(got), got, len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("source[%q] = %#v, want %#v", k, got[k], v)
		}
	}
}

func TestFlagToSource_NilCommand(t *testing.T) {
	if got := FlagToSource(nil); len(got) != 0 {
		t.Fatalf("FlagToSource(nil) = %v, want empty", got)
	}
}

func TestParseString_InvalidJSON(t *testing.T) {
	if _, err := ParseString([]byte("{")); !errors.Is(err, ErrParseSource) {
		t.Fatalf("err = %v, want ErrParseSource", err)
	}
}

func TestParseFile_NotExist(t *testing.T) {
	if _, err := ParseFile("../../data/conf/not_exist.json"); !errors.Is(err, ErrReadSource) {
		t.Fatalf("err = %v, want ErrReadSource", err)
	}
}
