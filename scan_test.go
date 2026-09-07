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
	"os"
	"path/filepath"
	"testing"
	"time"
)

func Test_Scan_IdenticalSizeGrouping(t *testing.T) {
	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "file1.bin")
	file2 := filepath.Join(tempDir, "file2.bin")
	file3 := filepath.Join(tempDir, "file3.bin")

	data1 := bytes.Repeat([]byte("A"), 128)
	data2 := bytes.Repeat([]byte("B"), 128)
	data3 := bytes.Repeat([]byte("C"), 128)

	require.NoError(t, os.WriteFile(file1, data1, 0600))
	require.NoError(t, os.WriteFile(file2, data2, 0600))
	require.NoError(t, os.WriteFile(file3, data3, 0600))

	groups, err := sylvred.Scan(tempDir, sylvred.ScanOptions{})
	require.NoError(t, err)

	require.Len(t, groups, 1)
	group := groups[0]

	require.Equal(t, int64(128), group.Size)
	require.Len(t, group.Entries, 3)
	require.Equal(t, int64(256), group.PotentialSavings())
	require.Len(t, group.Duplicates(), 2)
}

func Test_Scan_RejectionOfZeroByteFiles(t *testing.T) {
	tempDir := t.TempDir()

	empty1 := filepath.Join(tempDir, "empty1.dat")
	empty2 := filepath.Join(tempDir, "empty2.dat")
	empty3 := filepath.Join(tempDir, "empty3.dat")

	require.NoError(t, os.WriteFile(empty1, []byte{}, 0600))
	require.NoError(t, os.WriteFile(empty2, []byte{}, 0600))
	require.NoError(t, os.WriteFile(empty3, []byte{}, 0600))

	groups, err := sylvred.Scan(tempDir, sylvred.ScanOptions{})
	require.NoError(t, err)

	require.Empty(t, groups, "zero-byte files must never form candidate groups")
}

func Test_Scan_RejectionOfAlreadyHardLinkedFiles(t *testing.T) {
	tempDir := t.TempDir()

	master := filepath.Join(tempDir, "master.bin")
	link := filepath.Join(tempDir, "master_link.bin")

	data := bytes.Repeat([]byte("D"), 256)
	require.NoError(t, os.WriteFile(master, data, 0600))

	err := os.Link(master, link)
	if err != nil {
		t.Skipf("filesystem at %s does not support hard links: %v", tempDir, err)
	}

	groups, err := sylvred.Scan(tempDir, sylvred.ScanOptions{})
	require.NoError(t, err)

	require.Empty(t, groups, "files sharing identical inode must be rejected as non-candidates")
}

func Test_Scan_MixedHardLinkedAndSeparateInodes(t *testing.T) {
	tempDir := t.TempDir()

	master := filepath.Join(tempDir, "shared1.bin")
	link := filepath.Join(tempDir, "shared2.bin")
	independent := filepath.Join(tempDir, "independent.bin")

	data := bytes.Repeat([]byte("E"), 300)
	require.NoError(t, os.WriteFile(master, data, 0600))

	err := os.Link(master, link)
	if err != nil {
		t.Skipf("filesystem at %s does not support hard links: %v", tempDir, err)
	}

	require.NoError(t, os.WriteFile(independent, data, 0600))

	groups, err := sylvred.Scan(tempDir, sylvred.ScanOptions{})
	require.NoError(t, err)

	require.Len(t, groups, 1)
	group := groups[0]

	require.Equal(t, int64(300), group.Size)
	require.Len(t, group.Entries, 2, "must contain exactly 1 representative per distinct inode")
	require.Equal(t, int64(300), group.PotentialSavings())

	masterEntry := group.Master()
	require.Equal(t, uint64(2), masterEntry.LinkCount(), "entry with higher link count must be selected as master")
}

func Test_Scan_RecursiveSubdirectories(t *testing.T) {
	tempDir := t.TempDir()

	subDir := filepath.Join(tempDir, "sub", "deep")
	require.NoError(t, os.MkdirAll(subDir, 0700))

	fileA := filepath.Join(tempDir, "root.bin")
	fileB := filepath.Join(subDir, "deep.bin")

	data := bytes.Repeat([]byte("R"), 64)
	require.NoError(t, os.WriteFile(fileA, data, 0600))
	require.NoError(t, os.WriteFile(fileB, data, 0600))

	groups, err := sylvred.Scan(tempDir, sylvred.ScanOptions{})
	require.NoError(t, err)

	require.Len(t, groups, 1)
	require.Len(t, groups[0].Entries, 2)
}

func Test_Scan_UniqueSizeFilesNotGrouped(t *testing.T) {
	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "size10.bin")
	file2 := filepath.Join(tempDir, "size20.bin")

	require.NoError(t, os.WriteFile(file1, bytes.Repeat([]byte("1"), 10), 0600))
	require.NoError(t, os.WriteFile(file2, bytes.Repeat([]byte("2"), 20), 0600))

	groups, err := sylvred.Scan(tempDir, sylvred.ScanOptions{})
	require.NoError(t, err)

	require.Empty(t, groups)
}

func Test_CandidateGroup_MasterSelectionPrecedence(t *testing.T) {
	tempDir := t.TempDir()

	pathA := filepath.Join(tempDir, "a.bin")
	pathB := filepath.Join(tempDir, "b.bin")
	pathC := filepath.Join(tempDir, "c.bin")

	data := []byte("test-data-precedence")
	require.NoError(t, os.WriteFile(pathA, data, 0600))
	require.NoError(t, os.WriteFile(pathB, data, 0600))
	require.NoError(t, os.WriteFile(pathC, data, 0600))

	// Adjust mtime so pathB is oldest
	now := time.Now()
	tOld := now.Add(-10 * time.Minute)
	tNew := now.Add(-1 * time.Minute)
	require.NoError(t, os.Chtimes(pathB, tOld, tOld))
	require.NoError(t, os.Chtimes(pathA, tNew, tNew))
	require.NoError(t, os.Chtimes(pathC, tNew, tNew))

	groups, err := sylvred.Scan(tempDir, sylvred.ScanOptions{})
	require.NoError(t, err)
	require.Len(t, groups, 1)

	// Since all have link count 1, oldest mtime (pathB) must win
	require.Equal(t, pathB, groups[0].Master().Path())
}

/* ///////////////////////////// end of file //////////////////////////// */
