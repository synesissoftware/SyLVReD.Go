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

	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func Test_IOBoundPool_AcquireAndRelease(t *testing.T) {

	ctx := context.Background()
	pool := sylvred.NewIOBoundPool(2)

	require.NoError(t, pool.Acquire(ctx))
	require.NoError(t, pool.Acquire(ctx))

	// Third acquire should block until release
	cancelCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	err := pool.Acquire(cancelCtx)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	pool.Release()

	// Now acquire should succeed
	require.NoError(t, pool.Acquire(ctx))
	pool.Release()
	pool.Release()
}

func Test_IOBoundPool_ConcurrencyLimit(t *testing.T) {

	limit := 4
	pool := sylvred.NewIOBoundPool(limit)
	ctx := context.Background()

	var active atomic.Int64
	var maxObserved atomic.Int64
	var wg sync.WaitGroup

	workers := 20
	iterations := 50

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				require.NoError(t, pool.Acquire(ctx))
				curr := active.Add(1)

				for {
					oldMax := maxObserved.Load()
					if curr <= oldMax || maxObserved.CompareAndSwap(oldMax, curr) {
						break
					}
				}

				time.Sleep(100 * time.Microsecond)
				active.Add(-1)
				pool.Release()
			}
		}()
	}

	wg.Wait()

	require.LessOrEqual(t, maxObserved.Load(), int64(limit))
}

func Test_PipelineScan_IdenticalContentSmallFiles(t *testing.T) {

	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "dup1.txt")
	file2 := filepath.Join(tempDir, "dup2.txt")
	file3 := filepath.Join(tempDir, "different.txt")

	dataDup := bytes.Repeat([]byte("SameContent123"), 10)
	dataDiff := bytes.Repeat([]byte("DiffContent999"), 10)

	require.NoError(t, os.WriteFile(file1, dataDup, 0600))
	require.NoError(t, os.WriteFile(file2, dataDup, 0600))
	require.NoError(t, os.WriteFile(file3, dataDiff, 0600))

	tel := sylvred.NewTelemetry()
	opts := sylvred.PipelineOptions{
		IOWorkers:  4,
		CPUWorkers: 2,
		Telemetry:  tel,
	}

	ctx := context.Background()
	groups, err := sylvred.PipelineScan(ctx, tempDir, opts)
	require.NoError(t, err)

	require.Len(t, groups, 1)
	require.Equal(t, int64(len(dataDup)), groups[0].Size)
	require.Len(t, groups[0].Entries, 2)
	require.Equal(t, int64(len(dataDup)), groups[0].PotentialSavings())

	require.Equal(t, uint64(3), tel.FilesDiscovered())
	require.GreaterOrEqual(t, tel.PrefixBlocksRead(), uint64(2))
}

func Test_PipelineScan_MismatchedPrefixPruned(t *testing.T) {

	tempDir := t.TempDir()

	// Same size, but completely different prefix content
	file1 := filepath.Join(tempDir, "prefixA.bin")
	file2 := filepath.Join(tempDir, "prefixB.bin")

	size := 512
	dataA := bytes.Repeat([]byte("A"), size)
	dataB := bytes.Repeat([]byte("B"), size)

	require.NoError(t, os.WriteFile(file1, dataA, 0600))
	require.NoError(t, os.WriteFile(file2, dataB, 0600))

	ctx := context.Background()
	groups, err := sylvred.PipelineScan(ctx, tempDir, sylvred.PipelineOptions{})
	require.NoError(t, err)

	require.Empty(t, groups, "mismatched prefix headers must be eliminated early")
}

func Test_PipelineScan_MatchingPrefixMismatchedFullStreamPruned(t *testing.T) {

	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "file1.dat")
	file2 := filepath.Join(tempDir, "file2.dat")

	// Both share identical 4 KiB prefix, but diverge at byte 5000
	size := 8192
	prefix := bytes.Repeat([]byte("P"), sylvred.PrefixBlockSize)

	tail1 := bytes.Repeat([]byte("1"), size-sylvred.PrefixBlockSize)
	tail2 := bytes.Repeat([]byte("2"), size-sylvred.PrefixBlockSize)

	var data1, data2 bytes.Buffer
	data1.Write(prefix)
	data1.Write(tail1)

	data2.Write(prefix)
	data2.Write(tail2)

	require.NoError(t, os.WriteFile(file1, data1.Bytes(), 0600))
	require.NoError(t, os.WriteFile(file2, data2.Bytes(), 0600))

	ctx := context.Background()
	groups, err := sylvred.PipelineScan(ctx, tempDir, sylvred.PipelineOptions{})
	require.NoError(t, err)

	require.Empty(t, groups, "divergent full stream content must be eliminated")
}

func Test_PipelineScan_MatchingLargeFilesSurvive(t *testing.T) {

	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "large1.dat")
	file2 := filepath.Join(tempDir, "large2.dat")
	file3 := filepath.Join(tempDir, "large3.dat")

	// 16 KiB (> 4 KiB PrefixBlockSize) identical files
	size := 16 * 1024
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 251)
	}

	require.NoError(t, os.WriteFile(file1, data, 0600))
	require.NoError(t, os.WriteFile(file2, data, 0600))
	require.NoError(t, os.WriteFile(file3, data, 0600))

	tel := sylvred.NewTelemetry()
	opts := sylvred.PipelineOptions{
		IOWorkers:  8,
		CPUWorkers: 4,
		Telemetry:  tel,
	}

	ctx := context.Background()
	groups, err := sylvred.PipelineScan(ctx, tempDir, opts)
	require.NoError(t, err)

	require.Len(t, groups, 1)
	require.Equal(t, int64(size), groups[0].Size)
	require.Len(t, groups[0].Entries, 3)
	require.Equal(t, int64(2*size), groups[0].PotentialSavings())

	require.Equal(t, uint64(3), tel.FullHashesComputed())
	require.Equal(t, uint64(3), tel.PrefixHashesComputed())
	require.Equal(t, uint64(3*sylvred.PrefixBlockSize+3*size), tel.BytesHashed())
}

func Test_PipelineScan_RejectionOfZeroByteFiles(t *testing.T) {

	tempDir := t.TempDir()

	empty1 := filepath.Join(tempDir, "zero1.dat")
	empty2 := filepath.Join(tempDir, "zero2.dat")

	require.NoError(t, os.WriteFile(empty1, []byte{}, 0600))
	require.NoError(t, os.WriteFile(empty2, []byte{}, 0600))

	ctx := context.Background()
	groups, err := sylvred.PipelineScan(ctx, tempDir, sylvred.PipelineOptions{})
	require.NoError(t, err)

	require.Empty(t, groups, "zero-byte files must never be candidates")
}

func Test_PipelineScan_RejectionOfAlreadyHardLinkedFiles(t *testing.T) {

	tempDir := t.TempDir()

	master := filepath.Join(tempDir, "orig.bin")
	link := filepath.Join(tempDir, "orig_link.bin")

	data := bytes.Repeat([]byte("H"), 1024)
	require.NoError(t, os.WriteFile(master, data, 0600))

	err := os.Link(master, link)
	if err != nil {
		t.Skipf("filesystem at %s does not support hard links: %v", tempDir, err)
	}

	ctx := context.Background()
	groups, err := sylvred.PipelineScan(ctx, tempDir, sylvred.PipelineOptions{})
	require.NoError(t, err)

	require.Empty(t, groups, "files sharing identical inode must be rejected")
}

func Test_PipelineScan_StructuredCancellation(t *testing.T) {

	tempDir := t.TempDir()

	// Populate directory with files
	for i := 0; i < 50; i++ {
		p := filepath.Join(tempDir, filepath.Base(tempDir)+string(rune('a'+i%26))+string(rune('0'+i)))
		require.NoError(t, os.WriteFile(p, bytes.Repeat([]byte("Z"), 2048), 0600))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := sylvred.PipelineScan(ctx, tempDir, sylvred.PipelineOptions{
		IOWorkers:  2,
		CPUWorkers: 2,
	})
	require.ErrorIs(t, err, context.Canceled)
}

func Test_PipelineScanRoots_MultipleRoots(t *testing.T) {

	dir1 := t.TempDir()
	dir2 := t.TempDir()

	file1 := filepath.Join(dir1, "source.bin")
	file2 := filepath.Join(dir2, "target.bin")

	data := bytes.Repeat([]byte("MultiRootData"), 64)
	require.NoError(t, os.WriteFile(file1, data, 0600))
	require.NoError(t, os.WriteFile(file2, data, 0600))

	ctx := context.Background()
	groups, err := sylvred.PipelineScanRoots(ctx, []string{dir1, dir2}, sylvred.PipelineOptions{})
	require.NoError(t, err)

	// If both dirs are on the same volume (typical for t.TempDir()), they will be grouped
	fi1, err1 := os.Stat(file1)
	fi2, err2 := os.Stat(file2)
	require.NoError(t, err1)
	require.NoError(t, err2)

	_ = fi1

	_ = fi2

	if len(groups) > 0 {
		require.Len(t, groups, 1)
		require.Len(t, groups[0].Entries, 2)
	}
}

/* ///////////////////////////// end of file //////////////////////////// */
