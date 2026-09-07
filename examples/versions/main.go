// Copyright 2026 Matthew Wilson and Synesis Information Systems. All
// rights reserved. Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Created: 7th September 2026
 * Updated: 7th September 2026
 */

package main

import (
	angols "github.com/synesissoftware/ANGoLS"
	clasp "github.com/synesissoftware/CLASP.Go"
	diagnosticism "github.com/synesissoftware/Diagnosticism.Go"
	sylvred "github.com/synesissoftware/SyLVReD.Go"
	libclimate "github.com/synesissoftware/libCLImate.Go"
	libpath "github.com/synesissoftware/libpath.Go"
	p99 "github.com/synesissoftware/p99.Go"
	recls "github.com/synesissoftware/recls.Go"
	shwild "github.com/synesissoftware/shwild.Go"
	ver2go "github.com/synesissoftware/ver2go"
	woad "github.com/synesissoftware/woad.Go"

	"fmt"
)

func main() {

	fmt.Printf("SyLVReD.Go v%s\n", sylvred.VersionString())

	fmt.Println("depends on:")
	fmt.Printf("    ANGoLS v%s\n", angols.VersionString())
	fmt.Printf("    CLASP.Go v%s\n", clasp.VersionString())
	fmt.Printf("    Diagnosticism.Go v%s\n", diagnosticism.VersionString())
	fmt.Printf("    libCLImate.Go v%s\n", libclimate.VersionString())
	fmt.Printf("    libpath.Go v%s\n", libpath.VersionString())
	fmt.Printf("    p99.Go v%s\n", p99.VersionString())
	fmt.Printf("    recls.Go v%s\n", recls.VersionString())
	fmt.Printf("    shwild.Go v%s\n", shwild.VersionString())
	fmt.Printf("    ver2go v%s\n", ver2go.VersionString())
	fmt.Printf("    woad.Go v%s\n", woad.VersionString())
}

/* ///////////////////////////// end of file //////////////////////////// */
