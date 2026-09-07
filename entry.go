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
)

// Represents a file-system entry discovered during scanning. Embeds
// recls.Entry to provide full file-system metadata, path decomposition, and
// physical identity (DeviceId, Inode, LinkCount).
type FileEntry struct {
	recls.Entry
}

// Constructs a new FileEntry from a recls.Entry.
//
// Parameters:
//   - entry — the underlying file-system entry from recls;
func NewFileEntry(
	entry recls.Entry,
) FileEntry {

	return FileEntry{
		Entry: entry,
	}
}

// The device identifier on which the file resides (delegating to DeviceId).
func (fe FileEntry) Device() DeviceId {
	return DeviceId(fe.DeviceId())
}

/* ///////////////////////////// end of file //////////////////////////// */
