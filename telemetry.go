// Copyright 2026 Matthew Wilson and Synesis Information Systems. All
// rights reserved. Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Created: 7th September 2026
 * Updated: 7th September 2026
 */

package sylvred

import (
	p99 "github.com/synesissoftware/p99.Go"

	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Describes one active interval on the pipeline timeline.
type TimelineSpan struct {
	End   time.Duration
	Label string
	Start time.Duration
}

// Contains the search and processing intervals recorded for one pipeline run.
type TimelineSnapshot struct {
	Processing      []TimelineSpan
	ProcessingNames []string
	Search          []TimelineSpan
	SearchNames     []string
}

// Captures summary distribution statistics extracted from a p99.Histogram.
type DistributionStats struct {
	Count uint64
	P50   uint64
	P90   uint64
	P99   uint64
	Min   uint64
	Max   uint64
}

// Converts the P50 percentile value to a time.Duration.
func (s DistributionStats) P50Duration() time.Duration {
	return time.Duration(s.P50)
}

// Converts the P90 percentile value to a time.Duration.
func (s DistributionStats) P90Duration() time.Duration {
	return time.Duration(s.P90)
}

// Converts the P99 percentile value to a time.Duration.
func (s DistributionStats) P99Duration() time.Duration {
	return time.Duration(s.P99)
}

// Converts the minimum value to a time.Duration.
func (s DistributionStats) MinDuration() time.Duration {
	return time.Duration(s.Min)
}

// Converts the maximum value to a time.Duration.
func (s DistributionStats) MaxDuration() time.Duration {
	return time.Duration(s.Max)
}

// Manages real-time latency and throughput telemetry across the SyLVReD
// processing pipeline using p99.Go histograms.
type Telemetry struct {
	mu sync.Mutex

	traversalRateHist     *p99.Histogram
	traversalDurationHist *p99.Histogram
	prefixReadLatencyHist *p99.Histogram
	hashLatencyHist       *p99.Histogram
	hashThroughputHist    *p99.Histogram

	filesDiscovered          atomic.Uint64
	prefixBlocksRead         atomic.Uint64
	prefixHashesComputed     atomic.Uint64
	fullHashesComputed       atomic.Uint64
	bytesHashed              atomic.Uint64
	fileSearchDurationNs     atomic.Uint64
	fileProcessingDurationNs atomic.Uint64
	fileProcessingStartNs    atomic.Int64
	timelineOriginNs         atomic.Int64
	totalRuntimeNs           atomic.Uint64

	processingSpans []TimelineSpan
	processingNames []string
	searchSpans     []TimelineSpan
	searchNames     []string
}

// Constructs a new, fully initialized Telemetry manager.
func NewTelemetry() *Telemetry {

	return &Telemetry{
		traversalRateHist:     p99.New(),
		traversalDurationHist: p99.New(),
		prefixReadLatencyHist: p99.New(),
		hashLatencyHist:       p99.New(),
		hashThroughputHist:    p99.New(),
	}
}

// Increments the count of discovered files.
//
// Parameters:
//   - count — number of files to add to the running total;
func (t *Telemetry) AddDiscoveredFiles(
	count uint64,
) {

	if t == nil {
		return
	}

	t.filesDiscovered.Add(count)
}

// Returns the total number of files discovered during traversal.
func (t *Telemetry) FilesDiscovered() uint64 {

	if t == nil {
		return 0
	}

	return t.filesDiscovered.Load()
}

// Returns the total number of prefix blocks read.
func (t *Telemetry) PrefixBlocksRead() uint64 {

	if t == nil {
		return 0
	}

	return t.prefixBlocksRead.Load()
}

// Returns the total number of prefix block hashes computed.
func (t *Telemetry) PrefixHashesComputed() uint64 {

	if t == nil {
		return 0
	}

	return t.prefixHashesComputed.Load()
}

// Returns the total number of full stream hashes computed.
func (t *Telemetry) FullHashesComputed() uint64 {

	if t == nil {
		return 0
	}

	return t.fullHashesComputed.Load()
}

// Returns the total number of bytes streamed through hashing.
func (t *Telemetry) BytesHashed() uint64 {

	if t == nil {
		return 0
	}

	return t.bytesHashed.Load()
}

// Records the total end-to-end runtime for the pipeline execution.
//
// Records a wall-clock duration from pipeline entry to pipeline exit.
func (t *Telemetry) RecordTotalRuntime(
	d time.Duration,
) {

	if t == nil {
		return
	}

	if d < 0 {
		d = 0
	}

	ns := d.Nanoseconds()
	if ns < 0 {
		ns = 0
	}

	t.totalRuntimeNs.Store(uint64(ns))
}

// Returns the total pipeline execution runtime.
func (t *Telemetry) TotalRuntime() time.Duration {

	if t == nil {
		return 0
	}

	return time.Duration(t.totalRuntimeNs.Load())
}

// Sets the wall-clock origin used by subsequent timeline spans.
func (t *Telemetry) SetTimelineOrigin(
	origin time.Time,
) {

	if t == nil {
		return
	}

	t.timelineOriginNs.Store(origin.UnixNano())
}

// Records one search-goroutine activity interval.
func (t *Telemetry) RecordSearchSpan(
	label string,
	start time.Time,
	end time.Time,
) {

	if t == nil {
		return
	}

	t.recordTimelineSpan(&t.searchSpans, label, start, end)
}

// Registers a search goroutine for timeline reporting.
func (t *Telemetry) RegisterSearchWorker(
	label string,
) {

	if t == nil {
		return
	}

	t.registerTimelineWorker(&t.searchNames, label)
}

// Registers a processing goroutine for timeline reporting.
func (t *Telemetry) RegisterProcessingWorker(
	label string,
) {

	if t == nil {
		return
	}

	t.registerTimelineWorker(&t.processingNames, label)
}

// Records one processing-goroutine activity interval.
func (t *Telemetry) RecordProcessingSpan(
	label string,
	start time.Time,
	end time.Time,
) {

	if t == nil {
		return
	}

	t.recordTimelineSpan(&t.processingSpans, label, start, end)
}

// Returns a stable copy of the recorded search and processing intervals.
func (t *Telemetry) Timeline() TimelineSnapshot {

	if t == nil {
		return TimelineSnapshot{}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	snapshot := TimelineSnapshot{
		Processing:      append([]TimelineSpan(nil), t.processingSpans...),
		ProcessingNames: append([]string(nil), t.processingNames...),
		Search:          append([]TimelineSpan(nil), t.searchSpans...),
		SearchNames:     append([]string(nil), t.searchNames...),
	}
	sortTimelineSpans(snapshot.Processing)
	sortTimelineSpans(snapshot.Search)
	sort.Strings(snapshot.ProcessingNames)
	sort.Strings(snapshot.SearchNames)

	return snapshot
}

func (t *Telemetry) registerTimelineWorker(
	workers *[]string,
	label string,
) {

	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	for _, existing := range *workers {
		if existing == label {
			return
		}
	}

	*workers = append(*workers, label)
}

func (t *Telemetry) recordTimelineSpan(
	spans *[]TimelineSpan,
	label string,
	start time.Time,
	end time.Time,
) {

	if t == nil || end.Before(start) {
		return
	}

	originNs := t.timelineOriginNs.Load()
	if originNs == 0 {
		return
	}

	origin := time.Unix(0, originNs)
	startOffset := start.Sub(origin)
	endOffset := end.Sub(origin)
	if startOffset < 0 {
		startOffset = 0
	}
	if endOffset < startOffset {
		endOffset = startOffset
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	*spans = append(*spans, TimelineSpan{
		End:   endOffset,
		Label: label,
		Start: startOffset,
	})
}

func sortTimelineSpans(
	spans []TimelineSpan,
) {

	sort.SliceStable(spans, func(i int, j int) bool {
		if spans[i].Label != spans[j].Label {
			return spans[i].Label < spans[j].Label
		}
		return spans[i].Start < spans[j].Start
	})
}

// Records wall-clock time spent traversing the file system.
func (t *Telemetry) RecordFileSearchDuration(
	d time.Duration,
) {

	if t == nil {
		return
	}

	t.fileSearchDurationNs.Store(durationNanoseconds(d))
}

// Returns active time spent traversing the file system.
func (t *Telemetry) FileSearchDuration() time.Duration {

	if t == nil {
		return 0
	}

	return time.Duration(t.fileSearchDurationNs.Load())
}

// Starts the wall-clock file-processing interval.
func (t *Telemetry) BeginFileProcessing() {

	if t == nil {
		return
	}

	t.fileProcessingStartNs.CompareAndSwap(0, time.Now().UnixNano())
}

// Completes the wall-clock file-processing interval.
func (t *Telemetry) EndFileProcessing() {

	if t == nil {
		return
	}

	startNs := t.fileProcessingStartNs.Load()
	if startNs == 0 {
		return
	}

	t.RecordFileProcessingDuration(time.Since(time.Unix(0, startNs)))
}

// Records wall-clock time spent reading and hashing file contents.
func (t *Telemetry) RecordFileProcessingDuration(
	d time.Duration,
) {

	if t == nil {
		return
	}

	t.fileProcessingDurationNs.Store(durationNanoseconds(d))
}

// Returns active time spent reading and hashing file contents.
func (t *Telemetry) FileProcessingDuration() time.Duration {

	if t == nil {
		return 0
	}

	return time.Duration(t.fileProcessingDurationNs.Load())
}

// Records an instantaneous traversal rate in entries per second.
//
// Parameters:
//   - entriesPerSec — rate of entries discovered per second;
func (t *Telemetry) RecordTraversalRate(
	entriesPerSec uint64,
) {

	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.traversalRateHist.PushEventTimeNs(entriesPerSec)
}

// Records the duration of a traversal batch or directory traversal.
//
// Parameters:
//   - d — duration of the traversal event;
func (t *Telemetry) RecordTraversalDuration(
	d time.Duration,
) {

	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.traversalDurationHist.PushEventDuration(time.Duration(durationNanoseconds(d)))
}

// Records the latency of a 4 KiB prefix block read.
//
// Parameters:
//   - d — elapsed read duration;
func (t *Telemetry) RecordPrefixReadLatency(
	d time.Duration,
) {

	if t == nil {
		return
	}

	t.prefixBlocksRead.Add(1)

	t.mu.Lock()
	defer t.mu.Unlock()

	t.prefixReadLatencyHist.PushEventDuration(time.Duration(durationNanoseconds(d)))
}

// Records the latency of a hashing operation.
//
// Parameters:
//   - d — elapsed hashing duration;
func (t *Telemetry) RecordHashLatency(
	d time.Duration,
) {

	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.hashLatencyHist.PushEventDuration(time.Duration(durationNanoseconds(d)))
}

// Records the throughput of a hashing operation in bytes per second.
//
// Parameters:
//   - bytesPerSec — hashing speed in bytes per second;
func (t *Telemetry) RecordHashThroughput(
	bytesPerSec uint64,
) {

	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.hashThroughputHist.PushEventTimeNs(bytesPerSec)
}

// Records a prefix block hashing operation, updating latency, throughput,
// and prefix hash counters.
//
// Parameters:
//   - d — elapsed hashing duration;
//   - bytesProcessed — count of bytes processed during this operation;
func (t *Telemetry) RecordPrefixHash(
	d time.Duration,
	bytesProcessed int64,
) {

	if t == nil {
		return
	}

	if bytesProcessed > 0 {
		t.bytesHashed.Add(uint64(bytesProcessed))
		t.prefixHashesComputed.Add(1)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.hashLatencyHist.PushEventDuration(time.Duration(durationNanoseconds(d)))

	if d > 0 && bytesProcessed > 0 {
		ns := d.Nanoseconds()
		if ns > 0 {
			bytesPerSec := uint64((float64(bytesProcessed) * 1e9) / float64(ns))
			t.hashThroughputHist.PushEventTimeNs(bytesPerSec)
		}
	}
}

// Records a full stream hashing operation, updating latency, throughput,
// and full stream hash counters.
//
// Parameters:
//   - d — elapsed hashing duration;
//   - bytesProcessed — count of bytes processed during this operation;
func (t *Telemetry) RecordFullStreamHash(
	d time.Duration,
	bytesProcessed int64,
) {

	if t == nil {
		return
	}

	if bytesProcessed > 0 {
		t.bytesHashed.Add(uint64(bytesProcessed))
		t.fullHashesComputed.Add(1)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.hashLatencyHist.PushEventDuration(time.Duration(durationNanoseconds(d)))

	if d > 0 && bytesProcessed > 0 {
		ns := d.Nanoseconds()
		if ns > 0 {
			bytesPerSec := uint64((float64(bytesProcessed) * 1e9) / float64(ns))
			t.hashThroughputHist.PushEventTimeNs(bytesPerSec)
		}
	}
}

// Records a complete hashing operation, updating latency, throughput, and
// total byte counters. Delegated to RecordFullStreamHash.
//
// Parameters:
//   - d — elapsed hashing duration;
//   - bytesProcessed — count of bytes processed during this operation;
func (t *Telemetry) RecordHashOperation(
	d time.Duration,
	bytesProcessed int64,
) {

	t.RecordFullStreamHash(d, bytesProcessed)
}

// Extracts distribution statistics for traversal rate (entries per second).
func (t *Telemetry) TraversalRateStats() DistributionStats {

	if t == nil {
		return DistributionStats{}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	return extractStats(t.traversalRateHist)
}

// Extracts distribution statistics for traversal duration.
func (t *Telemetry) TraversalDurationStats() DistributionStats {

	if t == nil {
		return DistributionStats{}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	return extractStats(t.traversalDurationHist)
}

// Extracts distribution statistics for prefix block read latency.
func (t *Telemetry) PrefixReadLatencyStats() DistributionStats {

	if t == nil {
		return DistributionStats{}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	return extractStats(t.prefixReadLatencyHist)
}

// Extracts distribution statistics for hashing latency.
func (t *Telemetry) HashLatencyStats() DistributionStats {

	if t == nil {
		return DistributionStats{}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	return extractStats(t.hashLatencyHist)
}

// Extracts distribution statistics for hashing throughput (bytes per
// second).
func (t *Telemetry) HashThroughputStats() DistributionStats {

	if t == nil {
		return DistributionStats{}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	return extractStats(t.hashThroughputHist)
}

// Extracts distribution statistics from a p99.Histogram.
//
// When h is nil or empty, returns a zero-valued DistributionStats.
func extractStats(
	h *p99.Histogram,
) DistributionStats {

	if h == nil || h.EventCount() == 0 {
		return DistributionStats{}
	}

	_, p50 := h.ValueAtP50()
	_, p90 := h.ValueAtP90()
	_, p99 := h.ValueAtP99()
	_, minVal := h.MinEventTime()
	_, maxVal := h.MaxEventTime()

	return DistributionStats{
		Count: h.EventCount(),
		P50:   p50,
		P90:   p90,
		P99:   p99,
		Min:   minVal,
		Max:   maxVal,
	}
}

func durationNanoseconds(
	d time.Duration,
) uint64 {

	if d <= 0 {
		return 0
	}

	return uint64(d.Nanoseconds())
}

/* ///////////////////////////// end of file //////////////////////////// */
