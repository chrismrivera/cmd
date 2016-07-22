package cmd

import (
	"errors"
	"testing"
)

func TestCmdParse(t *testing.T) {
	testCases := []struct {
		cmdArgs   []string
		givenArgs []string
		success   bool
	}{
		{
			cmdArgs:   []string{"foo", "bar"},
			givenArgs: []string{},
			success:   false,
		},
		{
			cmdArgs:   []string{"foo", "bar"},
			givenArgs: []string{"a"},
			success:   false,
		},
		{
			cmdArgs:   []string{"foo", "bar"},
			givenArgs: []string{"a", "b"},
			success:   true,
		},
	}

	for i, tc := range testCases {
		c := NewCommand("test", "test-group", "does test stuff", nil, nil)

		for _, ca := range tc.cmdArgs {
			c.AppendArg(ca, ca)
		}

		if err := c.Parse(tc.givenArgs); (err == nil) != tc.success {
			t.Fatalf("Expected success: %t from test %d", tc.success, i)
		}
	}
}

func TestCmdArgs(t *testing.T) {
	testCases := []struct {
		args    []string
		success bool
	}{
		{
			args:    []string{"foo", "2", "true", "673758150012174337", "993758150012174337"},
			success: true,
		},
		{
			args:    []string{"foo", "dog", "true", "673758150012174337", "993758150012174337"},
			success: false,
		},
		{
			args:    []string{"foo", "2", "true", "-673758150012174337", "993758150012174337"},
			success: true,
		},
	}

	for i, tc := range testCases {
		c := NewCommand("test", "test-group", "does test stuff", nil, nil)
		c.AppendArg("a", "a string")
		c.AppendArg("b", "an int")
		c.AppendArg("c", "a bool")
		c.AppendArg("d", "an int64")
		c.AppendArg("e", "a uint64")

		if err := c.Parse(tc.args); err != nil {
			t.Fatal(err)
		}

		errs := []error{}

		// This can't really fail other than being empty
		if a := c.Arg("a").String(); a == "" {
			errs = append(errs, errors.New("string was empty"))
		}

		if _, err := c.Arg("b").Int(); err != nil {
			errs = append(errs, err)
		}

		if _, err := c.Arg("c").Bool(); err != nil {
			errs = append(errs, err)
		}

		if _, err := c.Arg("d").Int64(); err != nil {
			errs = append(errs, err)
		}

		if _, err := c.Arg("e").Uint64(); err != nil {
			errs = append(errs, err)
		}

		if tc.success && len(errs) != 0 {
			t.Fatalf("Expected success for test case %d: %v", i, errs)
		}

		if !tc.success && len(errs) == 0 {
			t.Fatalf("Expected failure, but got success for test case %d", i)
		}
	}
}

func TestCmdFlags(t *testing.T) {
	// String flag
	c := NewCommand("test", "test-group", "does test stuff", nil, nil)
	c.Flags.String("flag", "default", "description")

	if err := c.Parse([]string{"--flag", "trump"}); err != nil {
		t.Fatal(err)
	}

	if f := c.Flag("flag").String(); f != "trump" {
		t.Fatalf("Expected trump, got %s", f)
	}

	// String flag with default value
	c = NewCommand("test", "test-group", "does test stuff", nil, nil)
	c.Flags.String("flag", "default", "description")

	if err := c.Parse([]string{}); err != nil {
		t.Fatal(err)
	}

	if f := c.Flag("flag").String(); f != "default" {
		t.Fatalf("Expected default, got %s", f)
	}

	// Int flag
	c = NewCommand("test", "test-group", "does test stuff", nil, nil)
	c.Flags.Int("flag", 12, "description")

	if err := c.Parse([]string{"--flag", "20"}); err != nil {
		t.Fatal(err)
	}

	f, err := c.Flag("flag").Int()
	if err != nil {
		t.Fatal(err)
	}

	if f != 20 {
		t.Fatalf("Expected 20, got %d", f)
	}

	// Int flag with default value
	c = NewCommand("test", "test-group", "does test stuff", nil, nil)
	c.Flags.Int("flag", 12, "description")

	if err := c.Parse([]string{}); err != nil {
		t.Fatal(err)
	}

	f, err = c.Flag("flag").Int()
	if err != nil {
		t.Fatal(err)
	}

	if f != 12 {
		t.Fatalf("Expected 12, got %d", f)
	}

	// Bool flag
	c = NewCommand("test", "test-group", "does test stuff", nil, nil)
	c.Flags.Bool("flag", false, "description")

	if err := c.Parse([]string{"--flag"}); err != nil {
		t.Fatal(err)
	}

	fb, err := c.Flag("flag").Bool()
	if err != nil {
		t.Fatal(err)
	}

	if !fb {
		t.Fatalf("flag should have been true")
	}
}

func TestCmdVarArgs(t *testing.T) {
	c := NewCommand("test", "test-group", "does test stuff", nil, nil)
	c.AppendVarArg("names", "pet names")

	names := []string{"baxter", "dr. turner", "patrice"}

	if err := c.Parse(names); err != nil {
		t.Fatal(err)
	}

	for i, va := range c.VarArgs() {
		if va.String() != names[i] {
			t.Fatalf("%s != %s, position %d", va.String(), names[i], i)
		}
	}
}
