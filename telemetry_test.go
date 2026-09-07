// Copyright 2026 Matthew Wilson and Synesis Information Systems. All
// rights reserved. Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Created: 7th September 2026
 * Updated: 7th September 2026
 */

package sylvred_test

import (
	sylvred "github.com/synesissoftware/SyLVReD.Go"

	"github.com/stretchr/testify/require"

	"sync"
	"testing"
	"time"
)

func Test_Telemetry_NilSafety(t *testing.T) {

	var tel *sylvred.Telemetry

	require.NotPanics(t, func() {
		tel.AddDiscoveredFiles(10)
		require.Equal(t, uint64(0), tel.FilesDiscovered())
		require.Equal(t, uint64(0), tel.PrefixBlocksRead())
		require.Equal(t, uint64(0), tel.FullHashesComputed())
		require.Equal(t, uint64(0), tel.BytesHashed())

		tel.RecordTraversalRate(100)
		tel.RecordTraversalDuration(10 * time.Millisecond)
		tel.RecordFileSearchDuration(10 * time.Millisecond)
		tel.RecordFileProcessingDuration(10 * time.Millisecond)
		tel.RecordPrefixReadLatency(50 * time.Microsecond)
		tel.RecordHashLatency(100 * time.Microsecond)
		tel.RecordHashThroughput(50_000_000)
		tel.RecordHashOperation(200*time.Microsecond, 4096)

		require.Equal(t, uint64(0), tel.TraversalRateStats().Count)
		require.Equal(t, uint64(0), tel.TraversalDurationStats().Count)
		require.Equal(t, uint64(0), tel.PrefixReadLatencyStats().Count)
		require.Equal(t, uint64(0), tel.HashLatencyStats().Count)
		require.Equal(t, uint64(0), tel.HashThroughputStats().Count)
		require.Equal(t, time.Duration(0), tel.TotalRuntime())
		require.Equal(t, time.Duration(0), tel.FileSearchDuration())
		require.Equal(t, time.Duration(0), tel.FileProcessingDuration())
	})
}

func Test_Telemetry_RecordAndExtract_Latency(t *testing.T) {

	tel := sylvred.NewTelemetry()

	tel.RecordPrefixReadLatency(10 * time.Microsecond)
	tel.RecordPrefixReadLatency(20 * time.Microsecond)
	tel.RecordPrefixReadLatency(30 * time.Microsecond)

	require.Equal(t, uint64(3), tel.PrefixBlocksRead())

	stats := tel.PrefixReadLatencyStats()
	require.Equal(t, uint64(3), stats.Count)
	require.GreaterOrEqual(t, stats.P50, uint64(10_000))
	require.LessOrEqual(t, stats.P50, uint64(30_000))
	require.GreaterOrEqual(t, stats.P99, uint64(10_000))
	require.LessOrEqual(t, stats.P99, uint64(30_000))

	require.GreaterOrEqual(t, stats.P50Duration(), 10*time.Microsecond)
	require.LessOrEqual(t, stats.P50Duration(), 30*time.Microsecond)
}

func Test_Telemetry_RecordAndExtract_Throughput(t *testing.T) {

	tel := sylvred.NewTelemetry()

	// 1 MiB in 10 ms = 100 MiB/s = 104,857,600 bytes/sec
	tel.RecordHashOperation(10*time.Millisecond, 1024*1024)

	require.Equal(t, uint64(1), tel.FullHashesComputed())
	require.Equal(t, uint64(1024*1024), tel.BytesHashed())

	stats := tel.HashThroughputStats()
	require.Equal(t, uint64(1), stats.Count)
	require.Greater(t, stats.P50, uint64(100_000_000))

	latStats := tel.HashLatencyStats()
	require.Equal(t, uint64(1), latStats.Count)
	require.GreaterOrEqual(t, latStats.P50Duration(), 9*time.Millisecond)
}

func Test_Telemetry_RecordAndExtract_TotalRuntime(t *testing.T) {

	tel := sylvred.NewTelemetry()

	d := 123 * time.Millisecond
	tel.RecordTotalRuntime(d)

	require.Equal(t, d, tel.TotalRuntime())
}

func Test_Telemetry_RecordAndExtract_WorkDurations(t *testing.T) {

	tel := sylvred.NewTelemetry()

	searchDuration := 12 * time.Millisecond
	processingDuration := 34 * time.Millisecond

	tel.RecordFileSearchDuration(searchDuration)
	tel.RecordFileProcessingDuration(processingDuration)

	require.Equal(t, searchDuration, tel.FileSearchDuration())
	require.Equal(t, processingDuration, tel.FileProcessingDuration())
}

func Test_Telemetry_RecordAndExtract_Timeline(t *testing.T) {

	tel := sylvred.NewTelemetry()
	origin := time.Unix(100, 0)

	tel.SetTimelineOrigin(origin)
	tel.RegisterProcessingWorker("cpu-1")
	tel.RegisterSearchWorker("search-1")
	tel.RecordProcessingSpan("cpu-1", origin.Add(2*time.Millisecond), origin.Add(8*time.Millisecond))
	tel.RecordSearchSpan("search-1", origin, origin.Add(10*time.Millisecond))

	timeline := tel.Timeline()

	require.Equal(t, []string{"cpu-1"}, timeline.ProcessingNames)
	require.Equal(t, []string{"search-1"}, timeline.SearchNames)
	require.Equal(t, 1, len(timeline.Processing))
	require.Equal(t, 2*time.Millisecond, timeline.Processing[0].Start)
	require.Equal(t, 8*time.Millisecond, timeline.Processing[0].End)
	require.Equal(t, 1, len(timeline.Search))
	require.Equal(t, 10*time.Millisecond, timeline.Search[0].End)
}

func Test_Telemetry_RecordAndExtract_TraversalRate(t *testing.T) {

	tel := sylvred.NewTelemetry()

	tel.RecordTraversalRate(5000)
	tel.RecordTraversalRate(10000)
	tel.RecordTraversalRate(15000)

	stats := tel.TraversalRateStats()
	require.Equal(t, uint64(3), stats.Count)
	require.GreaterOrEqual(t, stats.P50, uint64(5000))
	require.LessOrEqual(t, stats.P50, uint64(15000))
}

func Test_Telemetry_ConcurrentSafety(t *testing.T) {

	tel := sylvred.NewTelemetry()

	var wg sync.WaitGroup
	workers := 16
	iterations := 500

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				d := time.Duration(100+workerId+j) * time.Microsecond
				tel.AddDiscoveredFiles(1)
				tel.RecordTraversalRate(uint64(1000 + j))
				tel.RecordTraversalDuration(d)
				tel.RecordPrefixReadLatency(d)
				tel.RecordHashLatency(d)
				tel.RecordHashThroughput(uint64(50_000_000 + j*1000))
				tel.RecordHashOperation(d, 4096)
			}
		}(i)
	}

	wg.Wait()

	totalOperations := uint64(workers * iterations)
	require.Equal(t, totalOperations, tel.FilesDiscovered())
	require.Equal(t, totalOperations, tel.PrefixBlocksRead())
	require.Equal(t, totalOperations, tel.FullHashesComputed())
	require.Equal(t, totalOperations*4096, tel.BytesHashed())

	pStats := tel.PrefixReadLatencyStats()
	require.Equal(t, totalOperations, pStats.Count)
}

/* ///////////////////////////// end of file //////////////////////////// */
