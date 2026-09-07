// Copyright 2026 Matthew Wilson and Synesis Information Systems. All
// rights reserved. Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Created: 7th September 2026
 * Updated: 7th September 2026
 */

package main

import (
	sylvred "github.com/synesissoftware/SyLVReD.Go"

	clasp "github.com/synesissoftware/CLASP.Go"
	d "github.com/synesissoftware/Diagnosticism.Go"
	libclimate "github.com/synesissoftware/libCLImate.Go"
	woad "github.com/synesissoftware/woad.Go"

	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

/* /////////////////////////////////////////////////////////////////////////
 * implementation functions
 */

// Formats a duration into a human-readable string with adaptive units. A
// zero duration formats as "—".
func formatDuration(
	dur time.Duration,
) string {

	if dur == 0 {
		return "—"
	}
	if dur < time.Microsecond {
		return fmt.Sprintf("%d ns", dur.Nanoseconds())
	} else if dur < time.Millisecond {
		return fmt.Sprintf("%.2f µs", float64(dur.Nanoseconds())/1000.0)
	} else if dur < time.Second {
		return fmt.Sprintf("%.2f ms", float64(dur.Microseconds())/1000.0)
	} else {
		return fmt.Sprintf("%.2f s", dur.Seconds())
	}
}

// Formats a per-second rate using decimal thousands separators. A zero rate
// formats as "—".
func formatRate(
	r uint64,
) string {

	if r == 0 {
		return "—"
	} else {
		return d.ItoaThousands(r) + " /s"
	}
}

// Formats a byte throughput value into an adaptive binary unit string. A
// zero throughput formats as "—".
func formatThroughput(
	b uint64,
) string {

	if b == 0 {
		return "—"
	}

	if b >= 1024*1024*1024 {
		return fmt.Sprintf("%.2f GiB/s", float64(b)/(1024*1024*1024))
	} else if b >= 1024*1024 {
		return fmt.Sprintf("%.2f MiB/s", float64(b)/(1024*1024))
	} else if b >= 1024 {
		return fmt.Sprintf("%.2f KiB/s", float64(b)/1024)
	} else {
		return fmt.Sprintf("%d B/s", b)
	}
}

/* /////////////////////////////////////////////////////////////////////////
 * main()
 */

func main() {

	flag_DryRun := clasp.Flag("--dry-run").SetAlias("-n").SetHelp("runs in dry-run mode without modifying the filesystem (default)")
	flag_Verbose := clasp.Flag("--verbose").SetAlias("-v").SetHelp("produces verbose output detailing candidate groups")
	option_VerboseMinSize := clasp.Option("--verbose-min-size").SetHelp("minimal file size in bytes for verbose candidate groups (default: 0)")
	option_VerboseMinLinks := clasp.Option("--verbose-min-links").SetHelp("minimal link count in any group member for verbose candidate groups (default: 2)")
	flag_ConcurrentSearch := clasp.Flag("--concurrent-search").SetHelp("runs each root search concurrently (default: false)")
	flag_Telemetry := clasp.Flag("--telemetry").SetHelp("displays latency and throughput percentiles (p50, p90, p99)")
	option_IOWorkers := clasp.Option("--io-workers").SetHelp("specifies maximum concurrent I/O reader workers (default: 32)")
	option_CPUWorkers := clasp.Option("--cpu-workers").SetHelp("specifies number of CPU hashing workers (default: runtime.GOMAXPROCS)")

	isDryRun := true
	isVerbose := false
	isConcurrentSearch := false
	isTelemetry := false
	verboseMinSizeBytes := int64(0)
	verboseMinLinks := uint64(2)
	ioWorkers := sylvred.DefaultIOWorkers
	cpuWorkers := runtime.GOMAXPROCS(0)

	climate, _ := libclimate.Init(func(cl *libclimate.Climate) error {

		cl.AddFlagFunc(flag_DryRun, func() {
			isDryRun = true
		})
		cl.AddFlagFunc(flag_Verbose, func() {
			isVerbose = true
		})
		cl.AddFlagFunc(flag_ConcurrentSearch, func() {
			isConcurrentSearch = true
		})
		cl.AddOptionFunc(option_VerboseMinSize, func(o *clasp.Argument, a *clasp.Specification) {

			_ = a

			n, parseErr := strconv.ParseInt(o.Value, 10, 64)
			if parseErr != nil || n < 0 {
				fmt.Fprintf(os.Stderr, "sylvred: invalid value for --verbose-min-size: %q\n", o.Value)
				os.Exit(1)
			}
			verboseMinSizeBytes = n
		})
		cl.AddOptionFunc(option_VerboseMinLinks, func(o *clasp.Argument, a *clasp.Specification) {

			_ = a

			n, parseErr := strconv.ParseUint(o.Value, 10, 64)
			if parseErr != nil {
				fmt.Fprintf(os.Stderr, "sylvred: invalid value for --verbose-min-links: %q\n", o.Value)
				os.Exit(1)
			}
			verboseMinLinks = n
		})
		cl.AddFlagFunc(flag_Telemetry, func() {
			isTelemetry = true
		})

		cl.AddOptionFunc(option_IOWorkers, func(o *clasp.Argument, a *clasp.Specification) {

			_ = a

			n, parseErr := strconv.Atoi(o.Value)
			if parseErr != nil || n < 1 {
				fmt.Fprintf(os.Stderr, "sylvred: invalid value for --io-workers: %q\n", o.Value)
				os.Exit(1)
			}
			ioWorkers = n
		})

		cl.AddOptionFunc(option_CPUWorkers, func(o *clasp.Argument, a *clasp.Specification) {

			_ = a

			n, parseErr := strconv.Atoi(o.Value)
			if parseErr != nil || n < 1 {
				fmt.Fprintf(os.Stderr, "sylvred: invalid value for --cpu-workers: %q\n", o.Value)
				os.Exit(1)
			}
			cpuWorkers = n
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

	r, _ := climate.ParseAndVerify(os.Args, libclimate.ParseFlag_PanicOnFailure)

	roots := make([]string, 0, len(r.Values))
	for _, val := range r.Values {
		roots = append(roots, val.Value)
	}

	run(
		r.ProgramName,
		roots,
		isDryRun,
		isVerbose,
		verboseMinSizeBytes,
		verboseMinLinks,
		ioWorkers,
		cpuWorkers,
		isTelemetry,
		isConcurrentSearch,
	)
}

// Coordinates a SyLVReD scan across the provided roots and renders the
// resulting candidate groups (and optional telemetry summaries) to stdout.
func run(
	programName string,
	roots []string,
	isDryRun bool,
	isVerbose bool,
	verboseMinSizeBytes int64,
	verboseMinLinks uint64,
	ioWorkers int,
	cpuWorkers int,
	isTelemetry bool,
	isConcurrentSearch bool,
) {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var tel *sylvred.Telemetry
	if isTelemetry {
		tel = sylvred.NewTelemetry()
	}

	opts := sylvred.PipelineOptions{
		ConcurrentSearch: isConcurrentSearch,
		CPUWorkers:       cpuWorkers,
		IOWorkers:        ioWorkers,
		Telemetry:        tel,
	}

	groups, err := sylvred.PipelineScanRoots(ctx, roots, opts)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintf(os.Stderr, "%s: operation cancelled\n", programName)
			os.Exit(130)
		}
		fmt.Fprintf(os.Stderr, "%s: scan failure: %v\n", programName, err)
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
	clrHeading := woad.FGBrightWhiteFor(os.Stdout)

	if isVerbose {
		for i, group := range groups {
			if group.Size < verboseMinSizeBytes {
				continue
			}

			if verboseMinLinks > 0 {
				hasEnoughLinks := false
				for _, member := range group.Entries {
					if member.LinkCount() >= verboseMinLinks {
						hasEnoughLinks = true
						break
					}
				}
				if !hasEnoughLinks {
					continue
				}
			}

			master := group.Master()
			duplicates := group.Duplicates()

			fmt.Printf("%sCandidate Group %d:%s\n", clrGroup, i+1, clrReset)
			fmt.Printf("  Group Size:        %s%d%s entries\n", clrCount, len(group.Entries), clrReset)
			fmt.Printf("  File Size:         %s%s%s bytes\n", clrSize, d.ItoaThousands(group.Size), clrReset)
			fmt.Printf("  Potential Savings: %s%s%s bytes\n", clrSavings, d.ItoaThousands(group.PotentialSavings()), clrReset)
			fmt.Printf("  Device:            %s0x%X%s\n", clrDevice, group.DeviceId, clrReset)
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
	fmt.Printf("  Search roots:            %s%s%s\n", clrGroup, d.ItoaThousands(len(roots)), clrReset)
	if isDryRun {
		fmt.Printf("  Execution mode:          %s%s%s\n", clrDryRun, "dry-run (no filesystem mutations performed)", clrReset)
	}
	fmt.Printf("  Discovered files:        %s%s%s\n", clrCount, d.ItoaThousands(tel.FilesDiscovered()), clrReset)
	fmt.Printf("  Candidate groups:        %s%s%s\n", clrGroup, d.ItoaThousands(len(groups)), clrReset)
	fmt.Printf("  Duplicate files:         %s%s%s\n", clrDuplicates, d.ItoaThousands(totalDuplicateFiles), clrReset)
	fmt.Printf("  Potential space savings: %s%s%s bytes\n", clrSize, d.ItoaThousands(totalReclaimableBytes), clrReset)

	if isTelemetry && tel != nil {
		travRate := tel.TraversalRateStats()
		prefRead := tel.PrefixReadLatencyStats()
		hashThrough := tel.HashThroughputStats()
		hashLat := tel.HashLatencyStats()

		fmt.Println()
		fmt.Printf("%sPipeline Telemetry Summary (p99.Go):%s\n", clrSummary, clrReset)
		fmt.Printf("  %s%-26s %-14s %-14s %-14s %10s%s\n",
			clrHeading, "Metric", "p50", "p90", "p99", "Samples", clrReset)
		fmt.Printf("  %s\n", "------------------------------------------------------------------------------------")

		fmt.Printf("  %-26s %-14s %-14s %-14s %10s\n",
			"Traversal rate",
			formatRate(travRate.P50),
			formatRate(travRate.P90),
			formatRate(travRate.P99),
			d.ItoaThousands(travRate.Count),
		)
		fmt.Printf("  %-26s %-14s %-14s %-14s %10s\n",
			"Prefix read latency",
			formatDuration(prefRead.P50Duration()),
			formatDuration(prefRead.P90Duration()),
			formatDuration(prefRead.P99Duration()),
			d.ItoaThousands(prefRead.Count),
		)
		fmt.Printf("  %-26s %-14s %-14s %-14s %10s\n",
			"Hashing throughput",
			formatThroughput(hashThrough.P50),
			formatThroughput(hashThrough.P90),
			formatThroughput(hashThrough.P99),
			d.ItoaThousands(hashThrough.Count),
		)
		fmt.Printf("  %-26s %-14s %-14s %-14s %10s\n",
			"Hashing latency",
			formatDuration(hashLat.P50Duration()),
			formatDuration(hashLat.P90Duration()),
			formatDuration(hashLat.P99Duration()),
			d.ItoaThousands(hashLat.Count),
		)

		fmt.Printf("  %s\n", "------------------------------------------------------------------------------------")
		fmt.Printf("  Prefix blocks read:      %s%s%s\n", clrCount, d.ItoaThousands(tel.PrefixBlocksRead()), clrReset)
		fmt.Printf("  Full stream hashes:      %s%s%s\n", clrCount, d.ItoaThousands(tel.FullHashesComputed()), clrReset)
		fmt.Printf("  Total bytes hashed:      %s%s%s bytes\n", clrSize, d.ItoaThousands(tel.BytesHashed()), clrReset)
		fmt.Printf("  Total runtime:           %s%s%s\n", clrSize, formatDuration(tel.TotalRuntime()), clrReset)
	}
}

/* ///////////////////////////// end of file //////////////////////////// */
