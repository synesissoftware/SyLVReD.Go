# SyLVReD.Go - TODO <!-- omit in toc -->


## Table of Contents <!-- omit in toc -->

* [Phase 2: v0.2.0 - Dual-Bound Concurrent Pipeline & Telemetry](#phase-2-v020---dual-bound-concurrent-pipeline--telemetry)
* [Phase 3: v0.3.0 - Byte Verification, Invariant Safety & Atomic Linking](#phase-3-v030---byte-verification-invariant-safety--atomic-linking)
* [Phase 4: v0.4.0 - Formatted Reporting, Benchmarks & Stress Testing](#phase-4-v040---formatted-reporting-benchmarks--stress-testing)


## Phase 2: v0.2.0 - Dual-Bound Concurrent Pipeline & Telemetry

* [x] Dual-bound staged concurrent pipeline (producer, candidate bucketer, I/O prefix worker pool, compute-bound hasher pool);
* [x] Configurable worker concurrency flags (`--io-workers`, `--cpu-workers`);
* [x] Real-time latency percentile telemetry ($p_{50}$, $p_{90}$, $p_{99}$) using **p99.Go**;
* [x] Channel backpressure buffering and structured context cancellation;
* [ ] Concurrency tuning and profiling for alpha2 milestone;


## Phase 3: v0.3.0 - Byte Verification, Invariant Safety & Atomic Linking

* [ ] Byte-for-byte stream verification eliminating cryptographic collision risk;
* [ ] Invariant assertions via **Diagnosticism.Go**;
* [ ] Wildcard path inclusion/exclusion pattern filtering via **shwild.Go**;
* [ ] Atomic link-and-replace filesystem mutation engine with adjacent temporary linking;


## Phase 4: v0.4.0 - Formatted Reporting, Benchmarks & Stress Testing

* [ ] ANSI SGR terminal formatting and TUI progress styling via **woad.Go**;
* [ ] Comprehensive benchmark suite comparing single-threaded baseline against concurrent pipeline;
* [ ] Concurrency stress testing under `go test -race ./...`;


<!-- ########################### end of file ########################### -->
