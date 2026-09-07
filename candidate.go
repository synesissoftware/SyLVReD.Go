// Copyright 2026 Matthew Wilson and Synesis Information Systems. All
// rights reserved. Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Created: 7th September 2026
 * Updated: 7th September 2026
 */

package sylvred

import (
	"cmp"
	"slices"
)

// Represents a group of candidate files sharing identical device and size
// attributes with differing inode identities.
type CandidateGroup struct {
	Device  uint64
	Size    int64
	Entries []FileEntry
}

// Returns the potential storage reclamation in bytes if duplicate files are
// consolidated into a single physical inode.
func (g CandidateGroup) PotentialSavings() int64 {

	if len(g.Entries) < 2 {
		return 0
	} else {
		return int64(len(g.Entries)-1) * g.Size
	}
}

// Selects the deterministic master entry for this candidate group according
// to the following precedence hierarchy:
//  1. Highest existing hard-link count (LinkCount);
//  2. Oldest modification timestamp (ModTime);
//  3. Lexicographically lowest canonical path string (Path).
func (g CandidateGroup) Master() FileEntry {

	if len(g.Entries) == 0 {
		return FileEntry{}
	}

	master := g.Entries[0]

	for i := 1; i < len(g.Entries); i++ {
		candidate := g.Entries[i]

		candLinks := candidate.LinkCount()
		mastLinks := master.LinkCount()

		if candLinks > mastLinks {
			master = candidate
		} else if candLinks < mastLinks {
			// retain current master
		} else {
			candMtime := candidate.ModTime()
			mastMtime := master.ModTime()

			if candMtime.Before(mastMtime) {
				master = candidate
			} else if mastMtime.Before(candMtime) {
				// retain current master
			} else {
				if candidate.Path() < master.Path() {
					master = candidate
				} else {
					// retain current master
				}
			}
		}
	}

	return master
}

// Returns all non-master entries in the candidate group, sorted
// lexicographically by path.
func (g CandidateGroup) Duplicates() []FileEntry {

	if len(g.Entries) < 2 {
		return nil
	}

	master := g.Master()
	duplicates := make([]FileEntry, 0, len(g.Entries)-1)

	for _, entry := range g.Entries {
		if entry.Path() != master.Path() {
			duplicates = append(duplicates, entry)
		}
	}

	slices.SortFunc(duplicates, func(a FileEntry, b FileEntry) int {
		return cmp.Compare(a.Path(), b.Path())
	})

	return duplicates
}

/* ///////////////////////////// end of file //////////////////////////// */
