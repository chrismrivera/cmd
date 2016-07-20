package cmd

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Arg struct {
	Name        string
	Description string
	Variable    bool
}

type Value string

func (v Value) String() string {
	return string(v)
}

func (v Value) Bool() (bool, error) {
	return strconv.ParseBool(string(v))
}

func (v Value) Int() (int, error) {
	i, err := strconv.ParseInt(string(v), 10, 32)
	return int(i), err
}

func (v Value) Int64() (int64, error) {
	return strconv.ParseInt(string(v), 10, 64)
}

func (v Value) Uint64() (uint64, error) {
	return strconv.ParseUint(string(v), 10, 64)
}

type SetupFunc func(cmd *Command)
type RunFunc func(cmd *Command) error

type Command struct {
	Name        string
	Description string
	Group       string
	Args        []*Arg
	EnvArgs     map[string]string
	Flags       *flag.FlagSet
	Setup       SetupFunc
	Run         RunFunc
}

func NewCommand(name, group, desc string, setup SetupFunc, run RunFunc) *Command {
	cmd := &Command{
		Name:        name,
		Description: desc,
		Group:       group,
		Flags:       flag.NewFlagSet(name, flag.ExitOnError),
		Args:        []*Arg{},
		EnvArgs:     map[string]string{},
	}

	cmd.Setup = setup
	cmd.Run = run

	return cmd
}

func (cmd *Command) AppendArg(name, desc string) {
	cmd.Args = append(cmd.Args, &Arg{name, desc, false})
}

func (cmd *Command) AppendVarArg(name, desc string) {
	cmd.Args = append(cmd.Args, &Arg{name, desc, true})
}

func (cmd *Command) AddFlag(name, defaultValue, desc string) {
	cmd.Flags.String(name, defaultValue, desc)
}

func (cmd *Command) AddFlagBool(name string, defaultValue bool, desc string) {
	cmd.Flags.Bool(name, defaultValue, desc)
}

func (cmd *Command) AddEnvArg(name, desc string) {
	cmd.EnvArgs[name] = desc
}

func (cmd *Command) Arg(name string) Value {
	for i, ca := range cmd.Args {
		if ca.Name == name {
			return Value(cmd.Flags.Arg(i))
		}
	}

	return ""
}

func (cmd *Command) EnvArg(name string) Value {
	return Value(strings.TrimSpace(os.Getenv(name)))
}

func (cmd *Command) VarArgs() []Value {
	ret := []Value{}

	for _, a := range cmd.Flags.Args()[len(cmd.Args)-1:] {
		ret = append(ret, Value(a))
	}

	return ret
}

func (cmd *Command) Flag(name string) Value {
	return Value(cmd.Flags.Lookup(name).Value.String())
}

func (cmd *Command) Parse(args []string) error {
	cmd.Flags.Parse(args)

	varArgs := false
	for _, arg := range cmd.Args {
		if arg.Variable {
			varArgs = true
			break
		}
	}

	if !varArgs && len(cmd.Flags.Args()) != len(cmd.Args) {
		return newUsageErr("Wrong number of command arguments", cmd.Usage)
	} else if varArgs && len(cmd.Flags.Args()) < len(cmd.Args) {
		return newUsageErr("Wrong number of command arguments", cmd.Usage)
	}

	if len(cmd.EnvArgs) > 0 {
		for n := range cmd.EnvArgs {
			if cmd.EnvArg(n) == "" {
				return newUsageErr(fmt.Sprintf("Environment variable %s is unset", n), cmd.Usage)
			}
		}
	}

	return nil
}

func (cmd *Command) Usage() {
	usageStr := ""
	cmdDesc := ""

	for _, a := range cmd.Args {
		if a.Variable {
			usageStr += a.Name + "... "
			cmdDesc += fmt.Sprintf("    %s[...]: %s\n", a.Name, a.Description)
		} else {
			usageStr += a.Name + " "
			cmdDesc += fmt.Sprintf("    %s: %s\n", a.Name, a.Description)
		}
	}

	fc := 0
	flagsStr := "Flags:\n"

	visitFunc := func(flag *flag.Flag) {
		flagsStr += fmt.Sprintf("    %s: %s\n", flag.Name, flag.Usage)
		fc++
	}

	cmd.Flags.VisitAll(visitFunc)

	usageflagStr := " [flags]"
	if fc == 0 {
		usageflagStr = ""
	}

	fmt.Printf("usage: %s %s%s %s\n\n", os.Args[0], cmd.Name, usageflagStr, usageStr)

	fmt.Printf("%s\n\n", cmd.Description)

	if len(cmd.Args) > 0 {
		fmt.Println("Command Arguments:")
		fmt.Println(cmdDesc)
	}

	if fc > 0 {
		fmt.Println(flagsStr)
	}

	if len(cmd.EnvArgs) > 0 {
		fmt.Println("Required environment variables:")

		for n, d := range cmd.EnvArgs {
			fmt.Printf("    %s: %s\n", n, d)
		}
	}
}

type UsageErr struct {
	errMsg    string
	showUsage func()
}

func (ue *UsageErr) Error() string {
	return ue.errMsg
}

func (ue *UsageErr) ShowUsage() {
	fmt.Println(ue.errMsg)
	fmt.Println()

	if ue.showUsage != nil {
		ue.showUsage()
	}
}

func newUsageErr(msg string, f func()) *UsageErr {
	if msg == "" {
		msg = "Invalid usage"
	}

	return &UsageErr{errMsg: msg, showUsage: f}
}

type App struct {
	Commands    map[string]*Command
	Description string
}

func NewApp() *App {
	return &App{
		Commands: make(map[string]*Command),
	}
}

func (app *App) AddCommand(cmd *Command) {
	app.Commands[cmd.Name] = cmd
	cmd.Setup(cmd)
}

func (app *App) Run(args []string) error {
	if len(args) < 2 {
		return newUsageErr("No command given", app.Usage)
	}

	if args[1] == "--help" {
		app.Usage()
		return nil
	}

	cmd, ok := app.Commands[args[1]]
	if !ok {
		return newUsageErr("Invalid command", app.Usage)
	}

	for _, arg := range args[2:] {
		if arg == "--help" {
			cmd.Usage()
			return nil
		}
	}

	if err := cmd.Parse(args[2:]); err != nil {
		return err
	}

	return cmd.Run(cmd)
}

func (app *App) Usage() {
	fmt.Printf("usage: %s cmd [cmd-flags] [cmd-args]\n", os.Args[0])

	if app.Description != "" {
		fmt.Println()
		fmt.Println(app.Description)
	}

	var groupNames sort.StringSlice
	cmdNamesByGroup := map[string]sort.StringSlice{}
	for _, cmd := range app.Commands {
		if _, ok := cmdNamesByGroup[cmd.Group]; !ok {
			groupNames = append(groupNames, cmd.Group)
		}

		cmdNamesByGroup[cmd.Group] = append(cmdNamesByGroup[cmd.Group], cmd.Name)
	}

	groupNames.Sort()

	for _, gn := range groupNames {
		fmt.Printf("\n%s:\n", gn)

		cmdNamesByGroup[gn].Sort()

		for _, cn := range cmdNamesByGroup[gn] {
			cmd := app.Commands[cn]
			fmt.Printf("    %-18s %s\n", cmd.Name, cmd.Description)
		}
	}

	fmt.Println()
}
