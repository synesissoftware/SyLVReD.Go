// Copyright 2026 Matthew Wilson and Synesis Information Systems. All
// rights reserved. Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Created: 7th September 2026
 * Updated: 7th September 2026
 */

package sylvred

import (
	libpath_parse "github.com/synesissoftware/libpath.Go/parse"
	recls "github.com/synesissoftware/recls.Go"

	"cmp"
	"os"
	"path/filepath"
	"slices"
)

// Options controlling the sequential filesystem traversal and candidate
// discovery.
type ScanOptions struct {
	// Patterns specifies optional file-name patterns to filter entries.
	// When empty, all files are matched ("*").
	Patterns []string
}

// Scans a single root directory recursively, discovering duplicate candidate
// groups partitioned by physical device and file size.
//
// Parameters:
//   - root — path to the directory hierarchy to traverse;
//   - opts — traversal and filtering options;
func Scan(
	root string,
	opts ScanOptions,
) ([]CandidateGroup, error) {

	return ScanRoots([]string{root}, opts)
}

// Scans one or more root directories recursively, aggregating files across
// all provided paths into candidate groups partitioned by physical device
// and file size.
//
// Parameters:
//   - roots — slice of directory paths to traverse;
//   - opts — traversal and filtering options;
func ScanRoots(
	roots []string,
	opts ScanOptions,
) ([]CandidateGroup, error) {

	buckets := make(map[uint64]map[int64][]FileEntry)

	searchFlags := recls.Files | recls.Recursive
	searchOpts := recls.SearchOptions{
		Flags: searchFlags,
	}

	cwd, cwdErr := os.Getwd()
	if cwdErr != nil {
		cwd = ""
	}

	for _, root := range roots {
		canonicalRoot, err := canonicalisePath(root, cwd)
		if err != nil {
			return nil, err
		}

		fi, err := os.Stat(canonicalRoot)
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			return nil, &os.PathError{
				Op:   "scan",
				Path: canonicalRoot,
				Err:  os.ErrInvalid,
			}
		}

		var searchSeq func(yield func(recls.Entry, error) bool)
		if len(opts.Patterns) == 0 {
			searchSeq = recls.Search(canonicalRoot, "*", searchOpts)
		} else {
			searchSeq = recls.Search(canonicalRoot, opts.Patterns, searchOpts)
		}

		for entry, searchErr := range searchSeq {
			if searchErr != nil {
				return nil, searchErr
			}

			if !entry.IsFile() {
				continue
			}

			// Invariant 3: Zero-byte files are not candidate entries.
			if entry.Size() == 0 {
				continue
			}

			fileEntry := NewFileEntry(entry)
			dev := fileEntry.DeviceId()

			deviceBuckets, exists := buckets[dev]
			if !exists {
				deviceBuckets = make(map[int64][]FileEntry)
				buckets[dev] = deviceBuckets
			}

			deviceBuckets[entry.Size()] = append(deviceBuckets[entry.Size()], fileEntry)
		}
	}

	return pruneAndOrderBuckets(buckets), nil
}

// Canonicalises a path string using libpath.Go and filepath.Clean.
func canonicalisePath(
	rawPath string,
	referenceDir string,
) (string, error) {

	pd, err := libpath_parse.ParsePathString(rawPath, referenceDir)
	if err != nil {
		return "", err
	}

	fullPath := pd.FullPath
	if fullPath == "" {
		fullPath = rawPath
	}

	return filepath.Clean(fullPath), nil
}

// Prunes buckets with fewer than 2 distinct inodes, selects canonical
// representatives per inode, and orders candidate groups deterministically.
func pruneAndOrderBuckets(
	buckets map[uint64]map[int64][]FileEntry,
) []CandidateGroup {

	var candidateGroups []CandidateGroup

	for dev, sizeBuckets := range buckets {
		for size, entries := range sizeBuckets {
			if len(entries) < 2 {
				continue
			}

			// Partition entries by Inode to identify already hard-linked files.
			inodeMap := make(map[uint64][]FileEntry)
			for _, entry := range entries {
				inodeMap[entry.Inode()] = append(inodeMap[entry.Inode()], entry)
			}

			// Invariant 2: If all entries share the same Inode, they are
			// already hard-linked duplicates and cannot be deduplicated further.
			if len(inodeMap) < 2 {
				continue
			}

			// Pick a deterministic representative FileEntry for each distinct Inode.
			groupEntries := make([]FileEntry, 0, len(inodeMap))
			for _, inodeEntries := range inodeMap {
				rep := selectInodeRepresentative(inodeEntries)
				groupEntries = append(groupEntries, rep)
			}

			// Sort entries within the group lexicographically by path.
			slices.SortFunc(groupEntries, func(a FileEntry, b FileEntry) int {
				return cmp.Compare(a.Path(), b.Path())
			})

			candidateGroups = append(candidateGroups, CandidateGroup{
				Device:  dev,
				Size:    size,
				Entries: groupEntries,
			})
		}
	}

	// Deterministic ordering of candidate groups:
	// 1. Device ascending;
	// 2. Size descending (largest potential savings first);
	// 3. Master path ascending lexicographically.
	slices.SortFunc(candidateGroups, func(a CandidateGroup, b CandidateGroup) int {
		if a.Device != b.Device {
			return cmp.Compare(a.Device, b.Device)
		} else if a.Size != b.Size {
			return cmp.Compare(b.Size, a.Size)
		} else {
			return cmp.Compare(a.Master().Path(), b.Master().Path())
		}
	})

	return candidateGroups
}

// Selects the deterministic representative for a set of entries that share
// the same Inode (preferring highest link count, oldest mtime, lowest path).
func selectInodeRepresentative(
	entries []FileEntry,
) FileEntry {

	if len(entries) == 0 {
		return FileEntry{}
	}

	rep := entries[0]
	for i := 1; i < len(entries); i++ {
		candidate := entries[i]

		candLinks := candidate.LinkCount()
		repLinks := rep.LinkCount()

		if candLinks > repLinks {
			rep = candidate
		} else if candLinks < repLinks {
			// retain current rep
		} else {
			candMtime := candidate.ModTime()
			repMtime := rep.ModTime()

			if candMtime.Before(repMtime) {
				rep = candidate
			} else if repMtime.Before(candMtime) {
				// retain current rep
			} else {
				if candidate.Path() < rep.Path() {
					rep = candidate
				} else {
					// retain current rep
				}
			}
		}
	}

	return rep
}

/* ///////////////////////////// end of file //////////////////////////// */
