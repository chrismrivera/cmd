package cmd

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
)

type CommandSetupFunc func(cmd *Command)
type CommandRunFunc func(cmd *Command) error

type CommandArg struct {
	Name        string
	Description string
}

type Command struct {
	Name        string
	Description string
	Group       string
	Args        []*CommandArg
	Flags       *flag.FlagSet
	Setup       CommandSetupFunc
	Run         CommandRunFunc
}

func NewCommand(name, group, desc string, setup CommandSetupFunc, run CommandRunFunc) *Command {
	cmd := &Command{
		Name:        name,
		Description: desc,
		Group:       group,
		Flags:       flag.NewFlagSet(name, flag.ExitOnError),
		Args:        []*CommandArg{},
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

func (cmd *Command) AppendArg(name, desc string) {
	cmd.Args = append(cmd.Args, &CommandArg{name, desc})
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

func (cmd *Command) Usage() {
	usageStr := ""
	cmdDesc := ""

	for _, a := range cmd.Args {
		usageStr += a.Name + " "
		cmdDesc += fmt.Sprintf("    %s: %s\n", a.Name, a.Description)
	}

	fc := 0
	flagsStr := "Flags:\n"

	visitFunc := func(flag *flag.Flag) {
		flagsStr += fmt.Sprintf("    --%s: %s\n", flag.Name, flag.Usage)
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
		fmt.Printf("Command Arguments:\n")
		fmt.Printf("%s\n", cmdDesc)
	}

	if fc > 0 {
		fmt.Printf(flagsStr)
	}

	os.Exit(0)
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
		cmdr.Usage()
	}

	cmd, ok := cmdr.Commands[args[1]]
	if !ok {
		cmdr.Usage()
	}

	for _, arg := range args[2:] {
		if arg == "--help" {
			cmd.Usage()
		}
	}

	cmd.Flags.Parse(args[2:])

	if len(cmd.Flags.Args()) != len(cmd.Args) {
		cmd.Usage()
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
	os.Exit(0)
}
