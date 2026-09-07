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
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func createBenchmarkTree(
	tb testing.TB,
	numDirs int,
	filesPerDir int,
	fileSize int,
	identical bool,
) string {

	tb.Helper()

	root := tb.TempDir()
	baseData := bytes.Repeat([]byte("BenchData1234567"), fileSize/16+1)
	baseData = baseData[:fileSize]

	for d := 0; d < numDirs; d++ {
		dir := filepath.Join(root, fmt.Sprintf("dir_%02d", d))
		require.NoError(tb, os.MkdirAll(dir, 0700))

		for f := 0; f < filesPerDir; f++ {
			filePath := filepath.Join(dir, fmt.Sprintf("file_%02d.bin", f))

			if identical {
				require.NoError(tb, os.WriteFile(filePath, baseData, 0600))
			} else {
				content := make([]byte, fileSize)
				copy(content, baseData)
				// differentiate prefix
				content[0] = byte((d*filesPerDir + f) % 256)
				content[1] = byte(((d*filesPerDir + f) / 256) % 256)
				require.NoError(tb, os.WriteFile(filePath, content, 0600))
			}
		}
	}

	return root
}

func Benchmark_Scan_vs_PipelineScan_IdenticalFiles(b *testing.B) {

	// 5 directories x 10 files = 50 duplicate files of 32 KiB each
	tree := createBenchmarkTree(b, 5, 10, 32*1024, true)
	ctx := context.Background()

	b.Run("Phase1_Sequential_Scan", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			groups, err := sylvred.Scan(tree, sylvred.ScanOptions{})
			if err != nil {
				b.Fatalf("Scan failed: %v", err)
			}
			if len(groups) == 0 {
				b.Fatal("expected candidate groups")
			}
		}
	})

	b.Run("Phase2_Concurrent_PipelineScan", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			groups, err := sylvred.PipelineScan(ctx, tree, sylvred.PipelineOptions{})
			if err != nil {
				b.Fatalf("PipelineScan failed: %v", err)
			}
			if len(groups) == 0 {
				b.Fatal("expected candidate groups")
			}
		}
	})
}

func Benchmark_Scan_vs_PipelineScan_DistinctFiles(b *testing.B) {

	// 5 directories x 10 files = 50 files of identical size but differing prefixes
	tree := createBenchmarkTree(b, 5, 10, 32*1024, false)
	ctx := context.Background()

	b.Run("Phase1_Sequential_Scan", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			groups, err := sylvred.Scan(tree, sylvred.ScanOptions{})
			if err != nil {
				b.Fatalf("Scan failed: %v", err)
			}
			// Phase 1 only checks size, so it groups all 50 files into 1 group
			if len(groups) == 0 {
				b.Fatal("expected candidate groups in Phase 1")
			}
		}
	})

	b.Run("Phase2_Concurrent_PipelineScan", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			groups, err := sylvred.PipelineScan(ctx, tree, sylvred.PipelineOptions{})
			if err != nil {
				b.Fatalf("PipelineScan failed: %v", err)
			}
			// Phase 2 eliminates differing prefixes, so 0 candidate groups survive
			if len(groups) != 0 {
				b.Fatal("expected 0 candidate groups in Phase 2 for distinct files")
			}
		}
	})
}

/* ///////////////////////////// end of file //////////////////////////// */
