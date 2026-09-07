module github.com/synesissoftware/SyLVReD.Go

go 1.24.0

require (
	github.com/stretchr/testify v1.12.1
	github.com/synesissoftware/ANGoLS v0.11.0
	github.com/synesissoftware/CLASP.Go v0.17.1
	github.com/synesissoftware/Diagnosticism.Go v0.13.4
	github.com/synesissoftware/libCLImate.Go v0.8.2-beta1
	github.com/synesissoftware/libpath.Go v0.0.2
	github.com/synesissoftware/p99.Go v0.2.1
	github.com/synesissoftware/recls.Go v0.1.0-alpha1
	github.com/synesissoftware/shwild.Go v0.2.8
	github.com/synesissoftware/ver2go v0.2.0
	github.com/synesissoftware/woad.Go v0.1.1
	golang.org/x/term v0.34.0
)

require (
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sys v0.35.0 // indirect
)

replace (
	github.com/synesissoftware/ANGoLS => ../../synesissoftware/freelibs/ANGoLS
	github.com/synesissoftware/CLASP.Go => ../../synesissoftware/freelibs/CLASP/CLASP.Go
	github.com/synesissoftware/Diagnosticism.Go => ../../synesissoftware/freelibs/Diagnosticism/Diagnosticism.Go
	github.com/synesissoftware/STEGoL => ../../synesissoftware/freelibs/STEGoL
	github.com/synesissoftware/libCLImate.Go => ../../synesissoftware/freelibs/libCLImate/libCLImate.Go
	github.com/synesissoftware/libpath.Go => ../../synesissoftware/freelibs/libpath/libpath.Go
	github.com/synesissoftware/p99.Go => ../../synesissoftware/freelibs/p99/p99.Go
	github.com/synesissoftware/recls.Go => ../../synesissoftware/freelibs/recls/recls.Go
	github.com/synesissoftware/shwild.Go => ../../synesissoftware/freelibs/shwild/shwild.Go
	github.com/synesissoftware/ver2go => ../../synesissoftware/freelibs/ver2go
	github.com/synesissoftware/woad.Go => ../../synesissoftware/freelibs/woad/woad.Go
)
