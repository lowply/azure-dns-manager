package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/google/go-cmp/cmp"
	yaml "gopkg.in/yaml.v2"
)

const (
	ExitCodeOk = iota
	ExitCodeParseFlagError
)

type CLI struct {
	outStream, errStream io.Writer
}

func (c *CLI) prep() error {
	if os.Getenv("AZURE_DNS_ZONES") == "" {
		return errors.New("AZURE_DNS_ZONES is empty")
	}

	if os.Getenv("AZURE_AUTH_LOCATION") == "" {
		return errors.New("AZURE_AUTH_LOCATION is empty")
	}

	_, err := os.Stat(os.Getenv("AZURE_DNS_ZONES"))
	if err != nil {
		fmt.Fprintln(c.errStream, "Wrong path for AZURE_DNS_ZONES")
		return err
	}

	_, err = os.Stat(os.Getenv("AZURE_AUTH_LOCATION"))
	if err != nil {
		fmt.Fprintln(c.errStream, "Wrong path for AZURE_AUTH_LOCATION")
		return err
	}

	session, err = NewAzureSession()
	if err != nil {
		return err
	}

	err = session.checkOrCreateResourceGroup()
	if err != nil {
		return err
	}

	return nil
}

func (c *CLI) getZone(zone string) error {
	z, err := NewZone(zone, true)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(z)
	if err != nil {
		return err
	}

	fmt.Println(string(data))

	return nil
}

func (c *CLI) createIfNot(zone string) error {
	exist := false

	list, err := session.listZones()
	if err != nil {
		return err
	}

	for _, v := range *list {
		if v == zone {
			exist = true
		}
	}

	if !exist {
		err = session.createZone(zone)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *CLI) syncZone(zone string) error {
	err := c.createIfNot(zone)
	if err != nil {
		return err
	}

	// Current zone
	remote, err := NewZone(zone, true)
	if err != nil {
		return err
	}

	// Ideal zone
	local, err := NewZone(zone, false)
	if err != nil {
		return err
	}

	// Compare entire zones first.
	if cmp.Equal(remote, local) {
		fmt.Println("No change")
		return nil
	}

	// Sync from local to remote.
	err = local.syncRecordSets(remote)
	if err != nil {
		return err
	}

	return nil
}

func (c *CLI) getNS(zone string) error {
	// Current zone
	_, err := NewZone(zone, true)
	if err != nil {
		return err
	}

	for _, v := range nsrecords {
		fmt.Println(v)
	}

	return nil
}

func (c *CLI) Run(args []string) int {
	err := c.prep()
	if err != nil {
		fmt.Fprintln(c.errStream, err)
		return ExitCodeParseFlagError
	}

	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(c.errStream)
	optSync := flags.String("s", "", "Sync a zone from the file to Azure DNS")
	optGet := flags.String("g", "", "Get a zone file from Azure DNS")
	optNS := flags.String("ns", "", "Get NS records for a domain")
	optHelp := flags.Bool("h", false, "Help message")

	err = flags.Parse(args[1:])
	if err != nil {
		return ExitCodeParseFlagError
	}

	if *optHelp {
		flags.Usage()
		return ExitCodeOk
	}

	// Disallow regular args
	if len(flags.Args()) > 0 {
		flags.Usage()
		return ExitCodeParseFlagError
	}

	// Usage
	if flags.NFlag() == 0 {
		flags.Usage()
		return ExitCodeOk
	}

	if *optGet != "" && *optSync != "" {
		fmt.Fprintln(c.errStream, "sync and get can't be used at once")
		return ExitCodeParseFlagError
	}

	if *optGet != "" {
		err := c.getZone(*optGet)
		if err != nil {
			fmt.Fprintln(c.errStream, err)
			return ExitCodeParseFlagError
		}
	}

	if *optSync != "" {
		err := c.syncZone(*optSync)
		if err != nil {
			fmt.Fprintln(c.errStream, err)
			return ExitCodeParseFlagError
		}
	}

	if *optNS != "" {
		err := c.getNS(*optNS)
		if err != nil {
			fmt.Fprintln(c.errStream, err)
			return ExitCodeParseFlagError
		}
	}

	return ExitCodeOk
}
