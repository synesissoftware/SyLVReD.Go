# SyLVReD.Go - Changes <!-- omit in toc -->


## 0.2.0-alpha1 - 7th September 2026

* implemented dual-bound staged concurrent pipeline (`PipelineScan`, `PipelineScanRoots`) with traversal producer, in-memory candidate size bucketer, bounded I/O worker pool, and CPU-pinned hash worker pool;
* implemented bounded concurrency token semaphore `IOBoundPool` limiting concurrent file open calls to prevent `EMFILE`/`ENFILE`;
* implemented 4 KiB prefix block equivalence pruning and full-stream SHA-256 hash refinement;
* added real-time latency percentile and throughput telemetry manager `Telemetry` instrumented via **p99.Go** ($p_{50}$, $p_{90}$, $p_{99}$);
* added CLI flags `--io-workers=<N>`, `--cpu-workers=<M>`, and `--telemetry` to `cmd/sylvred`;
* added telemetry percentile and throughput summary table formatted via **woad.Go** and **Diagnosticism.Go**;
* added comparative benchmark suite `pipeline_bench_test.go` evaluating Phase 1 sequential control against Phase 2 concurrent pipeline;
* added comprehensive test coverage validating structured context cancellation, concurrency limits, and equivalence pruning;


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
