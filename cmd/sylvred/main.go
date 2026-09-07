// Copyright 2026 Matthew Wilson and Synesis Information Systems. All
// rights reserved. Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Created: 7th September 2026
 * Updated: 7th September 2026
 */

package main

import (
	clasp "github.com/synesissoftware/CLASP.Go"
	d "github.com/synesissoftware/Diagnosticism.Go"
	sylvred "github.com/synesissoftware/SyLVReD.Go"
	libclimate "github.com/synesissoftware/libCLImate.Go"
	woad "github.com/synesissoftware/woad.Go"

	"fmt"
	"os"
)

func main() {

	flag_DryRun := clasp.Flag("--dry-run").SetAlias("-n").SetHelp("runs in dry-run mode without modifying the filesystem (default)")
	flag_Verbose := clasp.Flag("--verbose").SetAlias("-v").SetHelp("produces verbose output detailing candidate groups")

	isDryRun := true
	isVerbose := false

	climate, err := libclimate.Init(func(cl *libclimate.Climate) error {

		cl.AddFlagFunc(flag_DryRun, func() {
			isDryRun = true
		})
		cl.AddFlagFunc(flag_Verbose, func() {
			isVerbose = true
		})

		cl.ProgramName = "sylvred"
		cl.Version = sylvred.VersionString()

		cl.InfoLines = []string{
			"SyLVReD.Go - Systematic hard-Linking intra-Volume Recursive Deduplication",
			"",
			":version:",
			"",
		}

		cl.ValueNames = []string{"directory"}
		cl.ValuesConstraint = []int{1, -1}
		cl.ValuesString = "<directory> [ <directory> ... ]"

		return nil
	}, libclimate.InitFlag_PanicOnFailure)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sylvred: initialisation failure: %v\n", err)
		os.Exit(1)
	}

	r, err := climate.ParseAndVerify(os.Args, libclimate.ParseFlag_PanicOnFailure)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: argument parsing failure: %v\n", r.ProgramName, err)
		os.Exit(1)
	}

	roots := make([]string, 0, len(r.Values))
	for _, val := range r.Values {
		roots = append(roots, val.Value)
	}

	groups, err := sylvred.ScanRoots(roots, sylvred.ScanOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: scan failure: %v\n", r.ProgramName, err)
		os.Exit(1)
	}

	totalDuplicateFiles := 0
	var totalReclaimableBytes int64 = 0

	for _, group := range groups {
		totalDuplicateFiles += len(group.Duplicates())
		totalReclaimableBytes += group.PotentialSavings()
	}

	// colours

	clrReset := woad.ResetFor(os.Stdout)

	clrCount := woad.FGCyanFor(os.Stdout)
	clrSize := woad.FGBrightGreenFor(os.Stdout)

	clrDryRun := woad.FGBrightBlackFor(os.Stdout)
	clrDevice := woad.FGMagentaFor(os.Stdout)
	clrDuplicates := woad.FGBrightYellowFor(os.Stdout)
	clrGroup := woad.FGBrightCyanFor(os.Stdout)
	clrInode := woad.FGMagentaFor(os.Stdout)
	clrMTime := woad.FGBrightBlackFor(os.Stdout)
	clrSavings := woad.FGBrightGreenFor(os.Stdout)
	clrSummary := woad.FGBrightCyanFor(os.Stdout)

	if isVerbose {
		for i, group := range groups {
			master := group.Master()
			duplicates := group.Duplicates()

			fmt.Printf("%sCandidate Group %d:%s\n", clrGroup, i+1, clrReset)
			fmt.Printf("  Group Size:        %s%d%s entries\n", clrCount, len(group.Entries), clrReset)
			fmt.Printf("  File Size:         %s%s%s bytes\n", clrSize, d.ItoaThousands(group.Size), clrReset)
			fmt.Printf("  Potential Savings: %s%s%s bytes\n", clrSavings, d.ItoaThousands(group.PotentialSavings()), clrReset)
			fmt.Printf("  Device:            %s0x%X%s\n", clrDevice, group.Device, clrReset)
			fmt.Printf("  Master:            %s (links: %s%d%s, mtime: %s%s%s)\n",
				master.Path(),
				clrCount, master.LinkCount(), clrReset,
				clrMTime, master.ModTime().Format("2006-01-02 15:04:05"), clrReset,
			)
			fmt.Println("  Duplicates:")
			for _, dup := range duplicates {
				fmt.Printf("    %s (links: %s%d%s, inode: %s%d%s)\n",
					dup.Path(),
					clrCount, dup.LinkCount(), clrReset,
					clrInode, dup.Inode(), clrReset)
			}
			fmt.Println()
		}
	}

	fmt.Printf("%sSyLVReD Scan Summary:%s\n", clrSummary, clrReset)
	if isDryRun {
		fmt.Printf("  Execution mode:          %s%s%s\n", clrDryRun, "dry-run (no filesystem mutations performed)", clrReset)
	}
	fmt.Printf("  Candidate groups:        %s%d%s\n", clrGroup, len(groups), clrReset)
	fmt.Printf("  Duplicate files:         %s%d%s\n", clrDuplicates, totalDuplicateFiles, clrReset)
	fmt.Printf("  Potential space savings: %s%s%s bytes\n", clrSize, d.ItoaThousands(totalReclaimableBytes), clrReset)
}

/* ///////////////////////////// end of file //////////////////////////// */
