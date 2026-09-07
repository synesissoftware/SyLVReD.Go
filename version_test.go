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

	"testing"
)

const (
	Expected_VersionMajor uint16 = 0
	Expected_VersionMinor uint16 = 1
	Expected_VersionPatch uint16 = 0
	Expected_VersionAB    uint16 = 0xFFFF
)

func Test_Version_Elements(t *testing.T) {
	require.Equal(t, Expected_VersionMajor, sylvred.VersionMajor)
	require.Equal(t, Expected_VersionMinor, sylvred.VersionMinor)
	require.Equal(t, Expected_VersionPatch, sylvred.VersionPatch)
	require.Equal(t, Expected_VersionAB, sylvred.VersionAB)
}

func Test_Version(t *testing.T) {
	require.Equal(t, uint64(0x0000_0001_0000_FFFF), sylvred.Version())
}

func Test_VersionString(t *testing.T) {
	require.Equal(t, "0.1.0", sylvred.VersionString())
}

/* ///////////////////////////// end of file //////////////////////////// */
