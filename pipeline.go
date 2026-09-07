// Copyright 2026 Matthew Wilson and Synesis Information Systems. All
// rights reserved. Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Created: 7th September 2026
 * Updated: 7th September 2026
 */

package sylvred

import (
	recls "github.com/synesissoftware/recls.Go"

	"cmp"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"runtime"
	"slices"
	"sync"
	"time"
)

/* /////////////////////////////////////////////////////////////////////////
 * constants
 */

const (
	// Default number of concurrent I/O operations and reader workers.
	DefaultIOWorkers = 32

	// Size of the initial prefix block read for header equivalence checks.
	PrefixBlockSize = 4096
)

/* /////////////////////////////////////////////////////////////////////////
 * types
 */

// Options controlling concurrent pipeline execution, worker allocation, and
// telemetry instrumentation.
type PipelineOptions struct {
	// Optional file-name patterns to filter entries. When empty, all files
	// are matched ("*").
	Patterns []string

	// When enabled, each provided root directory is searched in its own
	// goroutine.
	ConcurrentSearch bool

	// Maximum number of concurrent I/O operations and I/O worker goroutines.
	// When <= 0, defaults to DefaultIOWorkers (32).
	IOWorkers int

	// Number of CPU-bound hashing workers. When <= 0, defaults to
	// runtime.GOMAXPROCS(0).
	CPUWorkers int

	// Receives execution metrics and latency histograms. Optional; when nil,
	// metrics collection is disabled.
	Telemetry *Telemetry
}

// Bounded concurrency token semaphore limiting concurrent file open calls
// to prevent exhausting operating system file descriptors (EMFILE/ENFILE).
type IOBoundPool struct {
	tokens chan struct{}
}

// Constructs a new IOBoundPool with the specified concurrency limit.
//
// Parameters:
//   - limit — maximum number of concurrent permits allowed;
func NewIOBoundPool(
	limit int,
) *IOBoundPool {

	if limit < 1 {
		limit = 1
	}

	return &IOBoundPool{
		tokens: make(chan struct{}, limit),
	}
}

// Acquires an I/O token permit, blocking until one is available or ctx is
// cancelled.
//
// Parameters:
//   - ctx — cancellation context;
func (p *IOBoundPool) Acquire(
	ctx context.Context,
) error {

	select {
	case p.tokens <- struct{}{}:

		return nil
	case <-ctx.Done():

		return ctx.Err()
	}
}

// Releases a previously acquired I/O token permit.
func (p *IOBoundPool) Release() {

	<-p.tokens
}

type prefixEntry struct {
	entry FileEntry
	data  []byte
}

type prefixGroup struct {
	deviceId DeviceId
	size     int64
	entries  []prefixEntry
}

/* /////////////////////////////////////////////////////////////////////////
 * API functions
 */

// Scans a single directory tree recursively using the staged concurrent
// pipeline, returning refined candidate groups whose members share identical
// size, prefix block hashes, and full stream SHA-256 digests.
//
// Parameters:
//   - ctx — cancellation context;
//   - root — path to the directory hierarchy to traverse;
//   - opts — pipeline configuration and concurrency options;
func PipelineScan(
	ctx context.Context,
	root string,
	opts PipelineOptions,
) ([]CandidateGroup, error) {

	return PipelineScanRoots(ctx, []string{root}, opts)
}

// Scans multiple directory trees recursively using the staged concurrent
// pipeline, returning refined candidate groups whose members share identical
// size, prefix block hashes, and full stream SHA-256 digests.
//
// Parameters:
//   - ctx — cancellation context;
//   - roots — slice of directory paths to traverse;
//   - opts — pipeline configuration and concurrency options;
func PipelineScanRoots(
	ctx context.Context,
	roots []string,
	opts PipelineOptions,
) ([]CandidateGroup, error) {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	start := time.Now()
	tel := opts.Telemetry
	tel.SetTimelineOrigin(start)
	defer func() {
		tel.RecordTotalRuntime(time.Since(start))
	}()
	defer tel.EndFileProcessing()

	ioWorkers := opts.IOWorkers
	if ioWorkers <= 0 {
		ioWorkers = DefaultIOWorkers
	}

	cpuWorkers := opts.CPUWorkers
	if cpuWorkers <= 0 {
		cpuWorkers = runtime.GOMAXPROCS(0)
	}
	if cpuWorkers < 1 {
		cpuWorkers = 1
	}

	ioPool := NewIOBoundPool(ioWorkers)

	var (
		pipelineErr error
		errOnce     sync.Once
	)

	setErr := func(err error) {
		if err != nil {
			errOnce.Do(func() {
				pipelineErr = err
				cancel()
			})
		}
	}

	// Canonicalise roots and verify directories before launching pipeline
	cwd, cwdErr := os.Getwd()
	if cwdErr != nil {
		cwd = ""
	}

	canonicalRoots, err := canonicalisePaths(roots, cwd)
	if err != nil {
		return nil, err
	}

	for _, cRoot := range canonicalRoots {
		fi, err := os.Stat(cRoot)
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			return nil, &os.PathError{
				Op:   "pipeline_scan",
				Path: cRoot,
				Err:  os.ErrInvalid,
			}
		}
	}

	// Inter-stage backpressure channels
	traversalChan := make(chan FileEntry, 1024)
	candidateChan := make(chan CandidateGroup, 256)
	prefixChan := make(chan prefixGroup, 512)
	survivorChan := make(chan CandidateGroup, 128)

	// Stage 1: Traversal Producer
	go runTraversalProducer(
		ctx,
		canonicalRoots,
		opts,
		traversalChan,
		setErr,
	)

	// Stage 2: Size & Device Bucketer
	go runSizeBucketer(
		ctx,
		traversalChan,
		candidateChan,
		opts.Telemetry,
	)

	// Stage 3: Bounded I/O Worker Pool
	var ioWg sync.WaitGroup
	for i := 0; i < ioWorkers; i++ {
		workerId := i + 1
		workerLabel := fmt.Sprintf("io-%d", workerId)
		tel.RegisterProcessingWorker(workerLabel)
		ioWg.Add(1)
		go func() {
			defer ioWg.Done()
			runIOWorker(
				ctx,
				workerLabel,
				candidateChan,
				prefixChan,
				ioPool,
				opts.Telemetry,
				setErr,
			)
		}()
	}

	go func() {
		ioWg.Wait()
		close(prefixChan)
	}()

	// Stage 4: CPU-Pinned Hash Worker Pool
	var cpuWg sync.WaitGroup
	for i := 0; i < cpuWorkers; i++ {
		workerId := i + 1
		workerLabel := fmt.Sprintf("cpu-%d", workerId)
		tel.RegisterProcessingWorker(workerLabel)
		cpuWg.Add(1)
		go func() {
			defer cpuWg.Done()
			runCPUWorker(
				ctx,
				workerLabel,
				prefixChan,
				survivorChan,
				ioPool,
				opts.Telemetry,
				setErr,
			)
		}()
	}

	go func() {
		cpuWg.Wait()
		close(survivorChan)
	}()

	// Stage 5: Fan-In Aggregation
	var allGroups []CandidateGroup
	for group := range survivorChan {
		allGroups = append(allGroups, group)
	}

	if pipelineErr != nil {
		return nil, pipelineErr
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Deterministic sorting of candidate groups and entries:
	// 1. Device ascending;
	// 2. Size descending;
	// 3. Master path ascending.
	for i := range allGroups {
		slices.SortFunc(allGroups[i].Entries, func(a FileEntry, b FileEntry) int {
			return cmp.Compare(a.Path(), b.Path())
		})
	}

	slices.SortFunc(allGroups, func(a CandidateGroup, b CandidateGroup) int {
		if a.DeviceId != b.DeviceId {
			return cmp.Compare(a.DeviceId, b.DeviceId)
		} else if a.Size != b.Size {
			return cmp.Compare(b.Size, a.Size)
		} else {
			return cmp.Compare(a.Master().Path(), b.Master().Path())
		}
	})

	return allGroups, nil
}

/* /////////////////////////////////////////////////////////////////////////
 * implementation functions
 */

// Partitions size/device buckets into inode-distinct candidate groups and
// emits them to the given channel (`out`) in no particular order.
//
// It prunes buckets where all entries already share a single inode (which
// indicates hard-linked duplicates) and selects one deterministic
// representative per distinct inode.
func emitSizeBuckets(
	ctx context.Context,
	buckets EntriesBySizeByDeviceId,
	out chan<- CandidateGroup,
) {

	for deviceId, sizeBuckets := range buckets {
		for size, entries := range sizeBuckets {
			if len(entries) < 2 {
				continue
			}

			inodeMap := make(map[InodeId][]FileEntry)
			for _, entry := range entries {
				inodeId := InodeId(entry.Inode())

				inodeMap[inodeId] = append(inodeMap[inodeId], entry)
			}

			// Invariant 2: If all entries share the same inode, they are
			// already hard-linked duplicates.
			if len(inodeMap) < 2 {
				continue
			}

			groupEntries := make([]FileEntry, 0, len(inodeMap))
			for _, inodeEntries := range inodeMap {
				rep := selectInodeRepresentative(inodeEntries)
				groupEntries = append(groupEntries, rep)
			}

			slices.SortFunc(groupEntries, func(a FileEntry, b FileEntry) int {
				return cmp.Compare(a.Path(), b.Path())
			})

			group := CandidateGroup{
				DeviceId: deviceId,
				Size:     size,
				Entries:  groupEntries,
			}

			select {
			case <-ctx.Done():

				return
			case out <- group:
			}
		}
	}
}

// Computes the SHA-256 digest of a file's full content stream.
//
// Parameters:
//   - ctx — cancellation context;
//   - path — file path to hash;
//   - size — file size in bytes (used for caller-side invariants);
//   - ioPool — bounded concurrency limiter for open/read operations;
//   - tel — optional telemetry sink.
func hashFullStream(
	ctx context.Context,
	path string,
	size int64,
	ioPool *IOBoundPool,
	tel *Telemetry,
) ([32]byte, error) {

	if err := ioPool.Acquire(ctx); err != nil {
		return [32]byte{}, err
	}
	defer ioPool.Release()

	start := time.Now()

	f, err := os.Open(path)
	if err != nil {
		return [32]byte{}, err
	}
	defer f.Close()

	hasher := sha256.New()
	buf := make([]byte, 64*1024)

	n, err := io.CopyBuffer(hasher, f, buf)
	if err != nil {
		return [32]byte{}, err
	}

	elapsed := time.Since(start)
	if tel != nil {
		tel.RecordHashOperation(elapsed, n)
	}

	var digest [32]byte
	copy(digest[:], hasher.Sum(nil))

	return digest, nil
}

// Groups candidate entries by full stream digest and emits CandidateGroups
// for digests that occur in at least two files.
//
// Parameters:
//   - ctx — cancellation context;
//   - device — device identifier for emitted groups;
//   - size — file size in bytes for emitted groups;
//   - candidates — entries eligible for full hashing;
//   - out — output channel for candidate groups;
//   - ioPool — bounded concurrency limiter for open/read operations;
//   - tel — optional telemetry sink;
//   - setErr — records the first pipeline error and cancels via caller.
func processFullStreamHashes(
	ctx context.Context,
	device DeviceId,
	size int64,
	candidates []FileEntry,
	out chan<- CandidateGroup,
	ioPool *IOBoundPool,
	tel *Telemetry,
	setErr func(error),
) {

	fullBuckets := make(map[[32]byte][]FileEntry)

	for _, entry := range candidates {
		select {
		case <-ctx.Done():

			return
		default:
		}

		digest, err := hashFullStream(ctx, entry.Path(), size, ioPool, tel)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			setErr(err)
			return
		}

		fullBuckets[digest] = append(fullBuckets[digest], entry)
	}

	for _, fullEntries := range fullBuckets {
		if len(fullEntries) < 2 {
			continue
		}

		group := CandidateGroup{
			DeviceId: device,
			Size:     size,
			Entries:  fullEntries,
		}

		select {
		case <-ctx.Done():

			return
		case out <- group:
		}
	}
}

// Refines prefix-based matches into final candidate groups.
//
// It hashes the prefix for each entry, partitions by prefix digest, and:
//   - if pg.size <= PrefixBlockSize, emits groups immediately;
//   - otherwise computes full stream hashes to eliminate mismatching tails.
func processPrefixGroup(
	ctx context.Context,
	pg prefixGroup,
	out chan<- CandidateGroup,
	ioPool *IOBoundPool,
	tel *Telemetry,
	setErr func(error),
) {

	prefixBuckets := make(map[[32]byte][]FileEntry)

	for _, pe := range pg.entries {
		start := time.Now()
		digest := sha256.Sum256(pe.data)
		elapsed := time.Since(start)

		if tel != nil {
			tel.RecordPrefixHash(elapsed, int64(len(pe.data)))
		}

		prefixBuckets[digest] = append(prefixBuckets[digest], pe.entry)
	}

	for _, candidateEntries := range prefixBuckets {
		if len(candidateEntries) < 2 {
			// Mismatched prefix header: eliminated early
			continue
		}

		if pg.size <= int64(PrefixBlockSize) {
			// Prefix covered the entire file; prefix hash is full digest
			group := CandidateGroup{
				DeviceId: pg.deviceId,
				Size:     pg.size,
				Entries:  candidateEntries,
			}

			select {
			case <-ctx.Done():

				return
			case out <- group:
			}
		} else {
			// Candidates match on size and prefix: compute full stream hashes
			processFullStreamHashes(
				ctx,
				pg.deviceId,
				pg.size,
				candidateEntries,
				out,
				ioPool,
				tel,
				setErr,
			)
		}
	}
}

// Constructs a prefixGroup by reading each candidate entry's prefix bytes.
//
// Entries whose prefix cannot be read are skipped; all others are retained
// with their associated prefix payload.
func readGroupPrefixes(
	ctx context.Context,
	group CandidateGroup,
	ioPool *IOBoundPool,
	tel *Telemetry,
) (prefixGroup, error) {

	pg := prefixGroup{
		deviceId: group.DeviceId,
		size:     group.Size,
		entries:  make([]prefixEntry, 0, len(group.Entries)),
	}

	prefixLen := int(group.Size)
	if prefixLen > PrefixBlockSize {
		prefixLen = PrefixBlockSize
	}

	for _, entry := range group.Entries {
		select {
		case <-ctx.Done():

			return prefixGroup{}, ctx.Err()
		default:
		}

		data, err := readPrefixBlock(ctx, entry.Path(), prefixLen, ioPool, tel)
		if err != nil {
			if ctx.Err() != nil {
				return prefixGroup{}, ctx.Err()
			}
			// Skip entries that cannot be read
			continue
		}

		pg.entries = append(pg.entries, prefixEntry{
			entry: entry,
			data:  data,
		})
	}

	return pg, nil
}

// Reads up to prefixLen bytes from the file at path and returns those
// bytes.
//
// It treats EOF and unexpected EOF as non-fatal (partial reads are
// allowed), while propagating other I/O failures. Telemetry is updated via
// tel when non-nil.
func readPrefixBlock(
	ctx context.Context,
	path string,
	prefixLen int,
	ioPool *IOBoundPool,
	tel *Telemetry,
) ([]byte, error) {

	if err := ioPool.Acquire(ctx); err != nil {
		return nil, err
	}
	defer ioPool.Release()

	start := time.Now()

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := make([]byte, prefixLen)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	elapsed := time.Since(start)
	if tel != nil {
		tel.RecordPrefixReadLatency(elapsed)
	}

	return buf[:n], nil
}

// CPU-bound worker that consumes prefixGroup values and emits refined
// CandidateGroups.
//
// It terminates when ctx is cancelled or the upstream in channel is closed.
func runCPUWorker(
	ctx context.Context,
	workerLabel string,
	in <-chan prefixGroup,
	out chan<- CandidateGroup,
	ioPool *IOBoundPool,
	tel *Telemetry,
	setErr func(error),
) {

	for {
		select {
		case <-ctx.Done():

			return
		case pg, ok := <-in:

			if !ok {
				return // upstream channel closed and empty; terminate worker cleanly
			}

			workStart := time.Now()
			processPrefixGroup(ctx, pg, out, ioPool, tel, setErr)
			tel.RecordProcessingSpan(workerLabel, workStart, time.Now())
		}
	}
}

// I/O-bound worker that consumes CandidateGroups, reads prefix blocks for
// their members, and emits the resulting prefixGroups.
//
// It terminates when ctx is cancelled or the upstream in channel is closed.
func runIOWorker(
	ctx context.Context,
	workerLabel string,
	in <-chan CandidateGroup,
	out chan<- prefixGroup,
	ioPool *IOBoundPool,
	tel *Telemetry,
	setErr func(error),
) {

	for {
		select {
		case <-ctx.Done():

			return
		case group, ok := <-in:

			if !ok {
				return // upstream channel closed and empty; terminate worker cleanly
			}

			workStart := time.Now()
			if pg, err := readGroupPrefixes(ctx, group, ioPool, tel); err != nil {
				tel.RecordProcessingSpan(workerLabel, workStart, time.Now())
				if ctx.Err() != nil {
					return
				}

				setErr(err)
				return
			} else {

				if len(pg.entries) >= 2 {
					select {
					case <-ctx.Done():

						return
					case out <- pg:
					}
				}
			}
			tel.RecordProcessingSpan(workerLabel, workStart, time.Now())

		}
	}
}

// Traversal stage that consumes FileEntry values from in, buckets them by
// device and size, and emits size-level candidate groups to out.
//
// It closes out when upstream completion is observed.
func runSizeBucketer(
	ctx context.Context,
	in <-chan FileEntry,
	out chan<- CandidateGroup,
	tel *Telemetry,
) {

	defer close(out)

	buckets := make(EntriesBySizeByDeviceId)

	for {
		select {
		case <-ctx.Done():

			return
		case entry, ok := <-in:

			if !ok {
				tel.BeginFileProcessing()
				emitSizeBuckets(ctx, buckets, out)
				return // upstream channel closed and empty; terminate worker cleanly
			}

			deviceId := DeviceId(entry.DeviceId())
			devBuckets, exists := buckets[deviceId]
			if !exists {
				devBuckets = make(EntriesBySize)
				buckets[deviceId] = devBuckets
			}

			devBuckets[entry.Size()] = append(devBuckets[entry.Size()], entry)
		}
	}
}

// Traversal producer that walks each root directory recursively, applying
// optional pattern filtering and pushing candidate FileEntry values into
// out.
//
// It skips non-file entries and rejects zero-byte files (which are never
// candidate entries). Telemetry is updated as files are discovered.
func runTraversalProducer(
	ctx context.Context,
	roots []string,
	opts PipelineOptions,
	out chan<- FileEntry,
	setErr func(error),
) {

	start := time.Now()
	defer func() {
		close(out)
		opts.Telemetry.RecordFileSearchDuration(time.Since(start))
	}()

	searchFlags := recls.Files | recls.Recursive
	searchOpts := recls.SearchOptions{
		Flags: searchFlags,
	}

	runForRoot := func(workerLabel string, root string) {

		searchStart := time.Now()
		defer func() {
			opts.Telemetry.RecordSearchSpan(workerLabel, searchStart, time.Now())
		}()

		var searchSeq func(yield func(recls.Entry, error) bool)
		if len(opts.Patterns) == 0 {
			searchSeq = recls.Search(root, "*", searchOpts)
		} else {
			searchSeq = recls.Search(root, opts.Patterns, searchOpts)
		}

		batchStart := time.Now()
		batchCount := 0

		for entry, searchErr := range searchSeq {
			if searchErr != nil {
				setErr(searchErr)
				return
			}

			select {
			case <-ctx.Done():

				return
			default:
			}

			if !entry.IsFile() {
				continue
			}

			// Invariant 3: Zero-byte files are not candidate entries.
			if entry.Size() == 0 {
				continue
			}

			fileEntry := NewFileEntry(entry)

			select {
			case <-ctx.Done():

				return
			case out <- fileEntry:
			}

			batchCount++
			if opts.Telemetry != nil {
				opts.Telemetry.AddDiscoveredFiles(1)

				if batchCount >= 64 {
					elapsed := time.Since(batchStart)
					if elapsed > 0 {
						rate := uint64((float64(batchCount) * 1e9) / float64(elapsed.Nanoseconds()))
						opts.Telemetry.RecordTraversalRate(rate)
						opts.Telemetry.RecordTraversalDuration(elapsed)
					}
					batchStart = time.Now()
					batchCount = 0
				}
			}
		}

		if opts.Telemetry != nil && batchCount > 0 {
			elapsed := time.Since(batchStart)
			if elapsed > 0 {
				rate := uint64((float64(batchCount) * 1e9) / float64(elapsed.Nanoseconds()))
				opts.Telemetry.RecordTraversalRate(rate)
				opts.Telemetry.RecordTraversalDuration(elapsed)
			}
		}
	}

	if opts.ConcurrentSearch {

		var wg sync.WaitGroup
		for i, root := range roots {
			workerLabel := fmt.Sprintf("search-%d", i+1)
			opts.Telemetry.RegisterSearchWorker(workerLabel)
			wg.Add(1)
			go func(workerId int, r string) {
				defer wg.Done()
				runForRoot(fmt.Sprintf("search-%d", workerId), r)
			}(i+1, root)
		}

		wg.Wait()
	} else {

		for i, root := range roots {
			select {
			case <-ctx.Done():

				return
			default:
			}

			workerLabel := fmt.Sprintf("search-%d", i+1)
			opts.Telemetry.RegisterSearchWorker(workerLabel)
			runForRoot(workerLabel, root)
		}
	}
}

/* ///////////////////////////// end of file //////////////////////////// */
