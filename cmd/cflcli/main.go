package main

import (
	"cflsplit/internal/core"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "--version", "version":
			fmt.Println(core.AppName + " " + core.Version)
			return
		case "--license", "--licence":
			fmt.Print(core.LicenseText)
			return
		case "--help", "-h":
			fmt.Println(core.AppName + " - CLI")
			fmt.Println("Commands: split | inspect | verify | join | --version | --license")
			fmt.Println(core.PrimaryWebsite + " | " + core.DiscoveryWebsite)
			return
		}
	}
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, core.AppName+" - CLI: split | inspect | verify | join | --version | --license")
		os.Exit(2)
	}
	cmd := os.Args[1]
	f := flag.NewFlagSet(cmd, flag.ExitOnError)
	src := f.String("source", "", "Source file")
	out := f.String("output", "", "Output parent (split) or new file (join)")
	parts := f.String("parts", "", "Folder containing parts")
	encoding := f.String("encoding", "auto", "auto (default), base64, utf8 (strict), binary")
	max := f.Int64("max-bytes", 20_000_000, "Maximum complete part bytes, including header")
	quiet := f.Bool("quiet", false, "Suppress progress")
	f.Parse(os.Args[2:])
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	notify := func(p core.Progress) {
		if !*quiet && p.Message != "" {
			fmt.Fprintln(os.Stderr, p.Phase+": "+p.Message)
		}
	}
	var result interface{}
	var e error
	switch cmd {
	case "split":
		result, e = core.Split(ctx, core.Options{Source: *src, OutputParent: *out, Encoding: *encoding, MaxPartBytes: *max}, notify)
	case "inspect":
		result, e = core.Inspect(*parts)
	case "verify":
		result, e = core.Verify(ctx, *parts, notify)
	case "join":
		if *out == "" {
			e = fmt.Errorf("join requires --output")
		} else {
			result, e = core.Join(ctx, *parts, *out, notify)
		}
	default:
		e = fmt.Errorf("unknown command: %s", cmd)
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", e)
		os.Exit(1)
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}
