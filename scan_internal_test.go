// Copyright 2026 Matthew Wilson and Synesis Information Systems. All
// rights reserved. Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Created: 7th September 2026
 * Updated: 7th September 2026
 */

package sylvred

import (
	"github.com/stretchr/testify/require"
	recls "github.com/synesissoftware/recls.Go"

	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func makeTestFileEntry(
	t *testing.T,
	path string,
) FileEntry {

	t.Helper()

	e, err := recls.Stat(path, 0)
	require.NoError(t, err)

	return NewFileEntry(e)
}

func Test_CanonicalisePath_RelativeWithReferenceDir(t *testing.T) {

	tempDir := t.TempDir()
	relPath := filepath.Join("sub", "file.txt")
	expected := filepath.Clean(filepath.Join(tempDir, "sub", "file.txt"))

	actual, err := canonicalisePath(relPath, tempDir)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func Test_CanonicalisePath_RedundantSegments(t *testing.T) {

	tempDir := t.TempDir()
	relPath := "a/b/../b/./c.txt"
	expected := filepath.Clean(filepath.Join(tempDir, "a", "b", "c.txt"))

	actual, err := canonicalisePath(relPath, tempDir)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func Test_CanonicalisePath_AbsolutePath(t *testing.T) {

	tempDir := t.TempDir()
	absPath := filepath.Clean(tempDir)

	actual, err := canonicalisePath(absPath, "/some/other/ref/dir")
	require.NoError(t, err)
	require.Equal(t, absPath, actual)
}

func Test_CanonicalisePaths_MultiplePaths(t *testing.T) {

	tempDir := t.TempDir()
	input := []string{
		"first/../first/a.txt",
		"second/b.txt",
	}
	expected := []string{
		filepath.Clean(filepath.Join(tempDir, "first", "a.txt")),
		filepath.Clean(filepath.Join(tempDir, "second", "b.txt")),
	}

	actual, err := canonicalisePaths(input, tempDir)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func Test_CanonicalisePaths_EmptySlice(t *testing.T) {

	actual, err := canonicalisePaths(nil, "/some/ref/dir")
	require.NoError(t, err)
	require.Empty(t, actual)
}

func Test_CanonicalisePaths_FailureInvalidChars(t *testing.T) {

	input := []string{
		"valid/dir",
		"<invalid>",
	}

	actual, err := canonicalisePaths(input, "/some/ref/dir")
	require.Error(t, err)
	require.Nil(t, actual)
}

func Test_CanonicalisePath_FailureInvalidChars(t *testing.T) {

	invalidPaths := []string{
		"<invalid>",
		">out",
		"|pipe",
		"?wild",
		"*glob",
	}

	for _, invalid := range invalidPaths {
		t.Run(invalid, func(t *testing.T) {
			actual, err := canonicalisePath(invalid, "/some/ref/dir")
			require.Error(t, err)
			require.Empty(t, actual)
		})
	}
}

func Test_SelectInodeRepresentative_EmptyEntries(t *testing.T) {

	rep := selectInodeRepresentative(nil)
	require.Equal(t, FileEntry{}, rep)
}

func Test_SelectInodeRepresentative_SingleEntry(t *testing.T) {

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "single.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("single"), 0600))

	entry := makeTestFileEntry(t, filePath)
	rep := selectInodeRepresentative([]FileEntry{entry})

	require.Equal(t, entry.Path(), rep.Path())
}

func Test_SelectInodeRepresentative_PrecedenceLinkCount(t *testing.T) {

	tempDir := t.TempDir()
	path1 := filepath.Join(tempDir, "link1.bin")
	path2 := filepath.Join(tempDir, "link2.bin")
	independentPath := filepath.Join(tempDir, "indep.bin")

	data := []byte("shared-link-data")
	require.NoError(t, os.WriteFile(path1, data, 0600))

	err := os.Link(path1, path2)
	if err != nil {
		t.Skipf("hard links not supported: %v", err)
	}

	require.NoError(t, os.WriteFile(independentPath, data, 0600))

	linkedEntry := makeTestFileEntry(t, path1)
	singleEntry := makeTestFileEntry(t, independentPath)

	require.Equal(t, uint64(2), linkedEntry.LinkCount())
	require.Equal(t, uint64(1), singleEntry.LinkCount())

	// Candidate with higher link count must be selected
	rep := selectInodeRepresentative([]FileEntry{singleEntry, linkedEntry})
	require.Equal(t, linkedEntry.Path(), rep.Path())

	rep2 := selectInodeRepresentative([]FileEntry{linkedEntry, singleEntry})
	require.Equal(t, linkedEntry.Path(), rep2.Path())
}

func Test_SelectInodeRepresentative_PrecedenceOlderMtime(t *testing.T) {

	tempDir := t.TempDir()
	pathOld := filepath.Join(tempDir, "old.bin")
	pathNew := filepath.Join(tempDir, "new.bin")

	data := []byte("mtime-test")
	require.NoError(t, os.WriteFile(pathOld, data, 0600))
	require.NoError(t, os.WriteFile(pathNew, data, 0600))

	now := time.Now()
	tOld := now.Add(-10 * time.Minute)
	tNew := now.Add(-1 * time.Minute)

	require.NoError(t, os.Chtimes(pathOld, tOld, tOld))
	require.NoError(t, os.Chtimes(pathNew, tNew, tNew))

	entryOld := makeTestFileEntry(t, pathOld)
	entryNew := makeTestFileEntry(t, pathNew)

	rep := selectInodeRepresentative([]FileEntry{entryNew, entryOld})
	require.Equal(t, entryOld.Path(), rep.Path())

	rep2 := selectInodeRepresentative([]FileEntry{entryOld, entryNew})
	require.Equal(t, entryOld.Path(), rep2.Path())
}

func Test_SelectInodeRepresentative_PrecedenceLexicographicalPath(t *testing.T) {

	tempDir := t.TempDir()
	pathA := filepath.Join(tempDir, "a.bin")
	pathB := filepath.Join(tempDir, "b.bin")

	data := []byte("path-tie-break")
	require.NoError(t, os.WriteFile(pathA, data, 0600))
	require.NoError(t, os.WriteFile(pathB, data, 0600))

	now := time.Now()
	require.NoError(t, os.Chtimes(pathA, now, now))
	require.NoError(t, os.Chtimes(pathB, now, now))

	entryA := makeTestFileEntry(t, pathA)
	entryB := makeTestFileEntry(t, pathB)

	// Since link count and mtime match, lexicographically smaller path ("a.bin") wins
	rep := selectInodeRepresentative([]FileEntry{entryB, entryA})
	require.Equal(t, entryA.Path(), rep.Path())

	rep2 := selectInodeRepresentative([]FileEntry{entryA, entryB})
	require.Equal(t, entryA.Path(), rep2.Path())
}

func Test_PruneAndOrderBuckets_PrunesSingleEntry(t *testing.T) {

	tempDir := t.TempDir()
	path1 := filepath.Join(tempDir, "single.bin")
	require.NoError(t, os.WriteFile(path1, []byte("single"), 0600))

	entry := makeTestFileEntry(t, path1)

	buckets := EntriesBySizeByDeviceId{
		DeviceId(entry.DeviceId()): {
			entry.Size(): {entry},
		},
	}

	groups := pruneAndOrderBuckets(buckets)
	require.Empty(t, groups)
}

func Test_PruneAndOrderBuckets_PrunesIdenticalInodes(t *testing.T) {

	tempDir := t.TempDir()
	path1 := filepath.Join(tempDir, "file1.bin")
	path2 := filepath.Join(tempDir, "file2.bin")

	data := []byte("already-hard-linked")
	require.NoError(t, os.WriteFile(path1, data, 0600))

	err := os.Link(path1, path2)
	if err != nil {
		t.Skipf("hard links not supported: %v", err)
	}

	entry1 := makeTestFileEntry(t, path1)
	entry2 := makeTestFileEntry(t, path2)

	buckets := EntriesBySizeByDeviceId{
		DeviceId(entry1.DeviceId()): {
			entry1.Size(): {entry1, entry2},
		},
	}

	groups := pruneAndOrderBuckets(buckets)
	require.Empty(t, groups, "entries sharing the same inode must be pruned")
}

func Test_PruneAndOrderBuckets_DeduplicatesSameInodeRetainingDistinct(t *testing.T) {

	tempDir := t.TempDir()
	path1 := filepath.Join(tempDir, "shared1.bin")
	path2 := filepath.Join(tempDir, "shared2.bin")
	pathIndep := filepath.Join(tempDir, "indep.bin")

	data := []byte("partially-linked")
	require.NoError(t, os.WriteFile(path1, data, 0600))

	err := os.Link(path1, path2)
	if err != nil {
		t.Skipf("hard links not supported: %v", err)
	}

	require.NoError(t, os.WriteFile(pathIndep, data, 0600))

	entry1 := makeTestFileEntry(t, path1)
	entry2 := makeTestFileEntry(t, path2)
	entryIndep := makeTestFileEntry(t, pathIndep)

	buckets := EntriesBySizeByDeviceId{
		DeviceId(entry1.DeviceId()): {
			entry1.Size(): {entry1, entry2, entryIndep},
		},
	}

	groups := pruneAndOrderBuckets(buckets)
	require.Len(t, groups, 1)

	group := groups[0]
	require.Equal(t, entry1.Size(), group.Size)
	require.Len(t, group.Entries, 2, "must contain exactly 1 entry per distinct inode")
}

func Test_PruneAndOrderBuckets_OrderingPrecedence(t *testing.T) {

	tempDir := t.TempDir()

	fileSmallA := filepath.Join(tempDir, "small_a.bin")
	fileSmallB := filepath.Join(tempDir, "small_b.bin")
	fileLargeA := filepath.Join(tempDir, "large_a.bin")
	fileLargeB := filepath.Join(tempDir, "large_b.bin")

	require.NoError(t, os.WriteFile(fileSmallA, bytes.Repeat([]byte("S"), 50), 0600))
	require.NoError(t, os.WriteFile(fileSmallB, bytes.Repeat([]byte("S"), 50), 0600))
	require.NoError(t, os.WriteFile(fileLargeA, bytes.Repeat([]byte("L"), 500), 0600))
	require.NoError(t, os.WriteFile(fileLargeB, bytes.Repeat([]byte("L"), 500), 0600))

	eSmallA := makeTestFileEntry(t, fileSmallA)
	eSmallB := makeTestFileEntry(t, fileSmallB)
	eLargeA := makeTestFileEntry(t, fileLargeA)
	eLargeB := makeTestFileEntry(t, fileLargeB)

	deviceId := DeviceId(eSmallA.DeviceId())

	buckets := EntriesBySizeByDeviceId{
		deviceId: {
			50:  {eSmallA, eSmallB},
			500: {eLargeA, eLargeB},
		},
	}

	groups := pruneAndOrderBuckets(buckets)
	require.Len(t, groups, 2)

	// Groups ordered by Size descending: 500 first, then 50
	require.Equal(t, int64(500), groups[0].Size)
	require.Equal(t, int64(50), groups[1].Size)
}

/* ///////////////////////////// end of file //////////////////////////// */
