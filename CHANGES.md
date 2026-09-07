# SyLVReD.Go - Changes <!-- omit in toc -->


## 0.1.0 - 7th September 2026

* added core domain models `FileEntry` (embedding `recls.Entry`) and `CandidateGroup`;
* implemented sequential filesystem scanner and candidate size bucketing engine (`Scan`, `ScanRoots`);
* added CLI entry point `cmd/sylvred` with `--dry-run`, `--verbose`, `--help`, and `--version` options;
* added unit-test fixtures and suite establishing the sequential baseline;
* added root `build.sh` and `run_all_unit_tests.sh` helper scripts and `.sis/` project identity files;
* bumped version to `0.1.0`;


## 0.0.0 - 7th September 2026

* initial project scaffolding;
* wired local dependency graph in `go.mod` for ten Synesis libraries;
* established root `version.go` and `version_test.go` per `ver2go` 0.2+ conventions;
* added `examples/versions` runtime version diagnostic program;


<!-- ########################### end of file ########################### -->
