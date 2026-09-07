// Copyright 2026 Matthew Wilson and Synesis Information Systems. All
// rights reserved. Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Created: 7th September 2026
 * Updated: 7th September 2026
 */

package sylvred

// A physical or virtual filesystem device identifier.
type DeviceId uint64

// A file size in bytes.
type FileSize = int64

// A filesystem inode index number.
type InodeId uint64

// Candidate entries grouped by file size.
type EntriesBySize map[FileSize][]FileEntry

// Candidate entries grouped by device identifier and file size.
type EntriesBySizeByDeviceId map[DeviceId]EntriesBySize

/* ///////////////////////////// end of file //////////////////////////// */
