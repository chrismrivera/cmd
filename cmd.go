package cmd

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

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

type SetupFunc func(cmd *Command)
type RunFunc func(cmd *Command) error

type Arg struct {
	Name        string
	Description string
	Variable    bool
}

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

func (cmd *Command) AddFlag(name, defaultValue, desc string) {
	cmd.Flags.String(name, defaultValue, desc)
}

func (cmd *Command) AddFlagBool(name string, defaultValue bool, desc string) {
	cmd.Flags.Bool(name, defaultValue, desc)
}

func (cmd *Command) AddEnvArg(name, desc string) {
	cmd.EnvArgs[name] = desc
}

func (cmd *Command) AppendArg(name, desc string) {
	cmd.Args = append(cmd.Args, &Arg{name, desc, false})
}

func (cmd *Command) AppendVarArg(name, desc string) {
	cmd.Args = append(cmd.Args, &Arg{name, desc, true})
}

func (cmd *Command) Flag(name string) string {
	return cmd.Flags.Lookup(name).Value.String()
}

func (cmd *Command) FlagUint(name string) (uint, error) {
	val := cmd.Flag(name)

	i, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return uint(0), err
	}

	return uint(i), nil
}

func (cmd *Command) FlagInt64(name string) (int64, error) {
	val := cmd.Flag(name)

	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}

	return i, nil
}

func (cmd *Command) FlagBool(name string) bool {
	f := cmd.Flags.Lookup(name)
	return f.Value.(flag.Getter).Get().(bool)
}

func (cmd *Command) Arg(name string) string {
	for i, ca := range cmd.Args {
		if ca.Name == name {
			return cmd.Flags.Arg(i)
		}
	}

	return ""
}

func (cmd *Command) ArgInt64(name string) (int64, error) {
	val := cmd.Arg(name)

	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}

	return i, nil
}

func (cmd *Command) ArgBool(name string) (bool, error) {
	return strconv.ParseBool(cmd.Arg(name))
}

func (cmd *Command) EnvArg(name string) string {
	return strings.TrimSpace(os.Getenv(name))
}

func (cmd *Command) VarArgs() []string {
	return cmd.Flags.Args()[len(cmd.Args)-1:]
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

type Commander struct {
	Commands map[string]*Command
}

func NewCommander() *Commander {
	return &Commander{make(map[string]*Command)}
}

func (cmdr *Commander) AddCommand(cmd *Command) {
	cmdr.Commands[cmd.Name] = cmd
	cmd.Setup(cmd)
}

func (cmdr *Commander) Run(args []string) error {
	if len(args) < 2 {
		return newUsageErr("No command given", cmdr.Usage)
	}

	cmd, ok := cmdr.Commands[args[1]]
	if !ok {
		return newUsageErr("Invalid command", cmdr.Usage)
	}

	for _, arg := range args[2:] {
		if arg == "--help" {
			return newUsageErr("", cmd.Usage)
		}
	}

	cmd.Flags.Parse(args[2:])

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
		for n, _ := range cmd.EnvArgs {
			if cmd.EnvArg(n) == "" {
				return newUsageErr(fmt.Sprintf("Environment variable %s is unset", n), cmd.Usage)
			}
		}
	}

	return cmd.Run(cmd)
}

func (cmdr *Commander) Usage() {
	fmt.Printf("usage: %s cmd [cmd-flags] [cmd-args]\n", os.Args[0])

	var groupNames sort.StringSlice
	cmdNamesByGroup := map[string]sort.StringSlice{}
	for _, cmd := range cmdr.Commands {
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
			cmd := cmdr.Commands[cn]
			fmt.Printf("    %-18s %s\n", cmd.Name, cmd.Description)
		}
	}

	fmt.Println()
}
