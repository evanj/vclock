// Copyright 2014 Evan Jones. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/evanj/vclock"
)

type stringerString string

func (s stringerString) String() string {
	return string(s)
}

func exitIfErr(err error, prefix string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %s: %s\n", prefix, err.Error())
		os.Exit(1)
	}
}

func main() {
	format := flag.String("format", "", "If set, runs dot with this output format (-T flag)")
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Fprint(os.Stderr, "Error: Missing input argument: drawvclocks (input)\n")
		fmt.Fprint(os.Stderr, "  Reads vector clocks from input, writing dot (graphviz) format to stdout\n")
		fmt.Fprint(os.Stderr, "  Input format: One clock per line: (optional label) n1, n2, ...\n\n")
		flag.Usage()
		os.Exit(1)
	}
	inputFile := flag.Args()[0]

	input, err := os.Open(inputFile)
	exitIfErr(err, "os.Open")

	clocks, labels, err := vclock.Parse(input)
	exitIfErr(err, "parsing vector clocks")
	g := vclock.ClocksToGraph(clocks)
	// replace the labels in the graph
	for _, n := range g.Nodes() {
		nodeString := n.Value().String()
		l, ok := labels[nodeString]
		if ok {
			g.SetValue(n, stringerString(l+"\n"+nodeString))
		}
	}

	var output io.WriteCloser = os.Stdout
	var dot *exec.Cmd
	if *format != "" {
		dot = exec.Command("dot", "-T"+*format)
		dot.Stdout = os.Stdout
		dot.Stderr = os.Stderr
		output, err = dot.StdinPipe()
		exitIfErr(err, "dot.StdinPipe")
		err := dot.Start()
		exitIfErr(err, "starting dot")
	}

	g.WriteDot(output)
	err = output.Close()
	exitIfErr(err, "output.Close")

	if dot != nil {
		err := dot.Wait()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error from dot: %s\n", err.Error())
			// For errors returned by dot, use dot's error code
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					os.Exit(status.ExitStatus())
				}
			}
			// some other kind of error; exit code 1
			os.Exit(1)
		}
	}
}
