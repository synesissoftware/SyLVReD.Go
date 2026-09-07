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
	recls "github.com/synesissoftware/recls.Go"

	"github.com/stretchr/testify/require"

	"os"
	"path/filepath"
	"testing"
)

func Test_NewFileEntry_CapturesMetadata(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sample.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("sample-content"), 0600))

	var reclsEntry recls.Entry
	for entry, err := range recls.Search(tempDir, "sample.txt", recls.SearchOptions{Flags: recls.Files}) {
		require.NoError(t, err)
		reclsEntry = entry
		break
	}
	require.True(t, reclsEntry.Exists())

	fileEntry := sylvred.NewFileEntry(reclsEntry)

	require.NotZero(t, fileEntry.DeviceId())
	require.NotZero(t, fileEntry.Inode())
	require.Equal(t, fileEntry.DeviceId(), fileEntry.Device())
	require.Equal(t, int64(14), fileEntry.Size())
	require.Equal(t, "sample.txt", fileEntry.Name())
	require.Equal(t, uint64(1), fileEntry.LinkCount())
	require.True(t, fileEntry.IsFile())
}

func Test_CandidateGroup_PotentialSavings(t *testing.T) {
	emptyGroup := sylvred.CandidateGroup{
		Device:  1,
		Size:    100,
		Entries: nil,
	}
	require.Equal(t, int64(0), emptyGroup.PotentialSavings())

	singleGroup := sylvred.CandidateGroup{
		Device:  1,
		Size:    100,
		Entries: make([]sylvred.FileEntry, 1),
	}
	require.Equal(t, int64(0), singleGroup.PotentialSavings())

	tripleGroup := sylvred.CandidateGroup{
		Device:  1,
		Size:    100,
		Entries: make([]sylvred.FileEntry, 3),
	}
	require.Equal(t, int64(200), tripleGroup.PotentialSavings())
}

/* ///////////////////////////// end of file //////////////////////////// */
