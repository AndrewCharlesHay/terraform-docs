/*
Copyright 2021 The terraform-docs Authors.

Licensed under the MIT license (the "License"); you may not
use this file except in compliance with the License.

You may obtain a copy of the License at the LICENSE file in
the root directory of this source tree.
*/

package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/terraform-docs/terraform-docs/cmd"
	"github.com/terraform-docs/terraform-docs/internal/format"
	"github.com/terraform-docs/terraform-docs/internal/terraform"
	"github.com/terraform-docs/terraform-docs/pkg/print"
)

// These are practiaclly a copy/paste of https://github.com/spf13/cobra/blob/master/doc/md_docs.go
// The reason we've decided to bring them over and not use them directly from cobra module was
// that we wanted to inject custom "Example" section with generated output based on the "examples"
// folder.

var basedir = "/docs"
var formatdir = "/formats"

func main() {
	err := generate(cmd.NewCommand(), "", "FORMATS_GUIDE")
	if err != nil {
		log.Fatal(err)
	}
}

func generate(cmd *cobra.Command, subdir string, basename string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if c.Annotations["kind"] == "" || c.Annotations["kind"] != "formatter" {
			continue
		}
		b := strings.Replace(strings.Replace(c.CommandPath(), " ", "-", -1), "terraform-docs-", "", -1)
		if err := generate(c, formatdir, b); err != nil {
			return err
		}
	}

	filename := filepath.Join("."+basedir, subdir, basename+".md")
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	if _, err := io.WriteString(f, ""); err != nil {
		return err
	}
	if err := generateMarkdown(cmd, f); err != nil {
		return err
	}
	return nil
}

func generateMarkdown(cmd *cobra.Command, w io.Writer) error {
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	buf := new(bytes.Buffer)
	name := cmd.CommandPath()

	short := cmd.Short
	long := cmd.Long
	if len(long) == 0 {
		long = short
	}

	buf.WriteString("## " + name + "\n\n")
	buf.WriteString(short + "\n\n")
	buf.WriteString("### Synopsis\n\n")
	buf.WriteString(long + "\n\n")

	if cmd.Runnable() {
		buf.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.UseLine()))
	}

	if len(cmd.Example) > 0 {
		buf.WriteString("### Examples\n\n")
		buf.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.Example))
	}

	if err := printOptions(buf, cmd, name); err != nil {
		return err
	}

	if len(cmd.Commands()) == 0 {
		if err := printExample(buf, name); err != nil {
			return err
		}
	} else {
		if err := printSeeAlso(buf, cmd.Commands()); err != nil {
			return err
		}
	}

	if !cmd.DisableAutoGenTag {
		buf.WriteString("###### Auto generated by spf13/cobra on " + time.Now().Format("2-Jan-2006") + "\n")
	}
	_, err := buf.WriteTo(w)
	return err
}

func printOptions(buf *bytes.Buffer, cmd *cobra.Command, name string) error {
	flags := cmd.NonInheritedFlags()
	flags.SetOutput(buf)
	if flags.HasAvailableFlags() {
		buf.WriteString("### Options\n\n```\n")
		flags.PrintDefaults()
		buf.WriteString("```\n\n")
	}

	parentFlags := cmd.InheritedFlags()
	parentFlags.SetOutput(buf)
	if parentFlags.HasAvailableFlags() {
		buf.WriteString("### Options inherited from parent commands\n\n```\n")
		parentFlags.PrintDefaults()
		buf.WriteString("```\n\n")
	}
	return nil
}

func getFlags(name string) string {
	switch strings.Replace(name, "terraform-docs ", "", -1) {
	case "pretty":
		return " --no-color"
	}
	return ""
}

func printExample(buf *bytes.Buffer, name string) error {
	buf.WriteString("### Example\n\n")
	buf.WriteString("Given the [`examples`](/examples/) module:\n\n")
	buf.WriteString("```shell\n")
	buf.WriteString(fmt.Sprintf("%s%s ./examples/\n", name, getFlags(name)))
	buf.WriteString("```\n\n")
	buf.WriteString("generates the following output:\n\n")

	settings := print.NewSettings()
	settings.ShowColor = false
	options := &terraform.Options{
		Path:           "./examples",
		ShowHeader:     true,
		HeaderFromFile: "main.tf",
		SortBy: &terraform.SortBy{
			Name:     settings.SortByName,
			Required: settings.SortByRequired,
		},
	}

	name = strings.Replace(name, "terraform-docs ", "", -1)
	printer, err := format.Factory(name, settings)
	if err != nil {
		return err
	}
	tfmodule, err := terraform.LoadWithOptions(options)
	if err != nil {
		log.Fatal(err)
	}
	output, err := printer.Print(tfmodule, settings)
	if err != nil {
		return err
	}
	segments := strings.Split(output, "\n")
	for _, s := range segments {
		if s == "" {
			buf.WriteString("\n")
		} else {
			buf.WriteString(fmt.Sprintf("    %s\n", s))
		}
	}
	buf.WriteString("\n")
	return nil
}

func printSeeAlso(buf *bytes.Buffer, children []*cobra.Command) error {
	buf.WriteString("### SEE ALSO\n\n")
	for _, child := range children {
		if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
			continue
		}
		if child.Annotations["kind"] == "" || child.Annotations["kind"] != "formatter" {
			continue
		}
		cname := child.CommandPath()
		link := strings.Replace(strings.Replace(cname, " ", "-", -1)+".md", "terraform-docs-", "", -1)
		buf.WriteString(fmt.Sprintf("* [%s](%s%s/%s)\t - %s\n", cname, basedir, formatdir, link, child.Short))
		for _, c := range child.Commands() {
			if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
				continue
			}
			if c.Annotations["kind"] == "" || c.Annotations["kind"] != "formatter" {
				continue
			}
			cname := c.CommandPath()
			link := strings.Replace(strings.Replace(cname, " ", "-", -1)+".md", "terraform-docs-", "", -1)
			buf.WriteString(fmt.Sprintf("  * [%s](%s%s/%s)\t - %s\n", cname, basedir, formatdir, link, c.Short))
		}
	}
	buf.WriteString("\n")
	return nil
}
