/*
Copyright 2023

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

var (
	debug, _  = strconv.ParseBool(os.Getenv("DSP_DEBUG"))
	datasetRx = regexp.MustCompile(`^\s*(datasets-[gs]et-entry)\s+(.+)\s*$`)
)

type datasetGetPayload struct {
	Dataset string   `json:"dataset,omitempty"`
	Keys    []string `json:"keys,omitempty"`
}

type datasetSetPayload struct {
	Dataset string         `json:"dataset,omitempty"`
	Entries []datasetEntry `json:"entries"`
}

type datasetEntry struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value,omitempty"`
}

func main() {
	// The original vmtools program .
	vmtoolsd := getPathToRealVmtoolsd()

	// Assert the vmtoolsd program exists.
	if _, err := os.Stat(vmtoolsd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logArgs("os.Args", os.Args)

	var (
		// Common arguments to pass to all vmtoolsd commands.
		commonArgs []string

		// Arguments for possible calls to vmtoolsd --cmd "info-get KEY" and/or
		// vmtoolsd --cmd "info-set KEY VAL".
		guestinfoArgs []string
	)

	// Iterate over all the arguments, looking for the "--cmd" arg. If found,
	// check whether or not the command is for getting or setting a datasets
	// entry. If so, then translate the command into one or more equivalent
	// commands for getting/setting guestinfo data.
	for i := 0; i < len(os.Args); i++ {
		if i > 0 && os.Args[i] == "--cmd" && i < len(os.Args)-1 {
			if m := datasetRx.FindStringSubmatch(os.Args[i+1]); m != nil {
				switch m[1] {
				case "datasets-get-entry":
					var p datasetGetPayload
					if err := json.Unmarshal([]byte(m[2]), &p); err != nil {
						fmt.Fprintln(os.Stderr, err)
						os.Exit(1)
					}
					keys := getGuestInfoKeysForDatasetGetPayload(p)
					for j := range keys {
						guestinfoArgs = append(
							guestinfoArgs,
							fmt.Sprintf("info-get %s", keys[j]),
						)
					}
				case "datasets-set-entry":
					var p datasetSetPayload
					if err := json.Unmarshal([]byte(m[2]), &p); err != nil {
						fmt.Fprintln(os.Stderr, err)
						os.Exit(1)
					}
					kvp := getGuestInfoKeyValuePairsForDatasetSetPayload(p)
					for k, v := range kvp {
						guestinfoArgs = append(
							guestinfoArgs,
							fmt.Sprintf("info-set %s %s", k, v),
						)
					}
				}
				i++
			} else {
				commonArgs = append(commonArgs, os.Args[i])
			}
		} else {
			commonArgs = append(commonArgs, os.Args[i])
		}
	}

	logArgs("commonArgs", commonArgs)
	logArgs("guestinfoArgs", guestinfoArgs)

	switch len(guestinfoArgs) {
	case 0:
		execAndExit(vmtoolsd, commonArgs...)
	case 1:
		execAndExit(vmtoolsd, append(commonArgs, "--cmd", guestinfoArgs[0])...)
	default:
		for i := range guestinfoArgs {
			args := append(commonArgs[1:], "--cmd", guestinfoArgs[i])
			execAndExitOnError(vmtoolsd, args...)
		}
	}
}

func getPathToRealVmtoolsd() string {
	if v := os.Getenv("DSP_VMTOOLSD"); v != "" {
		return v
	}

	// Try to get the absolute, evaluated path to this program, dereferencing
	// any symlinks along the way.
	programPath, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	programPath, err = filepath.EvalSymlinks(programPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return programPath + ".bin"
}

func logArgs(msg string, args []string) {
	if !debug {
		return
	}
	log.Printf("%s len=%d, val=%s\n", msg, len(args), strings.Join(args, ","))
}

func execAndExitOnError(name string, args ...string) {
	logArgs("execAndExitOnError", args)

	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			os.Exit(err.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execAndExit(name string, args ...string) {
	logArgs("execAndExit", args)

	if runtime.GOOS == "windows" {
		// Windows does not support the *nix style of process forking, so just
		// invoke the Windows vmtoolsd binary as a separate process.
		execAndExitOnError(name, args[1:]...)
		os.Exit(0)
	}

	// All non-Windows operating systems support replacing the current process
	// via syscall.Exec.
	if err := syscall.Exec(name, args, os.Environ()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(0)
}

func getGuestInfoKeysForDatasetGetPayload(p datasetGetPayload) []string {
	keys := make([]string, len(p.Keys))
	for i := range p.Keys {
		keys[i] = getGuestInfoKeyForDatasetNameAndKey(p.Dataset, p.Keys[i])
	}
	return keys
}

func getGuestInfoKeyValuePairsForDatasetSetPayload(
	p datasetSetPayload) map[string]string {

	kvp := make(map[string]string, len(p.Entries))
	for i := range p.Entries {
		key := getGuestInfoKeyForDatasetNameAndKey(p.Dataset, p.Entries[i].Key)
		kvp[key] = string(p.Entries[i].Value)
	}
	return kvp
}

func getGuestInfoKeyForDatasetNameAndKey(name, key string) string {
	return fmt.Sprintf("guestinfo.%s.%s", name, key)
}
