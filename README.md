# SyLVReD.Go <!-- omit in toc -->

**Sy**stematic hard-**L**inking intra-**V**olume **Re**cursive **D**eduplication, in Go


[![Language](https://img.shields.io/badge/Language-Go-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)
[![GitHub release](https://img.shields.io/github/v/release/synesissoftware/SyLVReD.Go.svg)](https://github.com/synesissoftware/SyLVReD.Go/releases/latest)
[![Last Commit](https://img.shields.io/github/last-commit/synesissoftware/SyLVReD.Go)](https://github.com/synesissoftware/SyLVReD.Go/commits/master)
[![Go](https://github.com/synesissoftware/SyLVReD.Go/actions/workflows/go.yml/badge.svg)](https://github.com/synesissoftware/SyLVReD.Go/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/synesissoftware/SyLVReD.Go.svg)](https://pkg.go.dev/github.com/synesissoftware/SyLVReD.Go)


## Table of Contents <!-- omit in toc -->

- [Introduction](#introduction)
- [Installation](#installation)
- [Components](#components)
- [Examples](#examples)
- [Project Information](#project-information)
  - [Where to get help](#where-to-get-help)
  - [Contribution guidelines](#contribution-guidelines)
  - [Dependencies](#dependencies)
    - [Development / Testing Dependencies](#development--testing-dependencies)
  - [Related projects](#related-projects)
  - [License](#license)


## Introduction

**SyLVReD** is **Sy**stematic hard-**L**inking intra-**V**olume **Re**cursive **D**eduplication: a program that analyses file-system contents and equivalences, and uses intra-volume hard-linking to reduce storage cost. **SyLVReD.Go** is the **Go** implementation.


## Installation

Install via `go get`, as in:

```bash
go get "github.com/synesissoftware/SyLVReD.Go"
```


## Components

T.B.C.


## Examples

Examples are provided in the `examples` directory, along with a markdown description for each. A detailed TOC is in [EXAMPLES.md](./EXAMPLES.md).


## Project Information


### Where to get help

[GitHub Page](https://github.com/synesissoftware/SyLVReD.Go "GitHub Page")


### Contribution guidelines

Defect reports, feature requests, and pull requests are welcome on https://github.com/synesissoftware/SyLVReD.Go.

If you'd like to help out with the project, please raise an issue via [GitHub Page](https://github.com/synesissoftware/SyLVReD.Go/issues "GitHub Page") - you'll be very welcome!


### Dependencies

**SyLVReD.Go** depends on the following libraries:

* [**ANGoLS**](https://github.com/synesissoftware/ANGoLS/);
* [**CLASP.Go**](https://github.com/synesissoftware/CLASP.Go/);
* [**Diagnosticism.Go**](https://github.com/synesissoftware/Diagnosticism.Go/);
* [**libCLImate.Go**](https://github.com/synesissoftware/libCLImate.Go/);
* [**libpath.Go**](https://github.com/synesissoftware/libpath.Go/);
* [**p99.Go**](https://github.com/synesissoftware/p99.Go/);
* [**recls.Go**](https://github.com/synesissoftware/recls.Go/);
* [**shwild.Go**](https://github.com/synesissoftware/shwild.Go/);
* [**ver2go**](https://github.com/synesissoftware/ver2go/);
* [**woad.Go**](https://github.com/synesissoftware/woad.Go/);


#### Development / Testing Dependencies

* [**testify**](https://github.com/stretchr/testify);


### Related projects

* [**SyLVReD**](https://github.com/synesissoftware/SyLVReD/);
* [**SyLVReD.Ruby**](https://github.com/synesissoftware/SyLVReD.Ruby/);
* [**SyLVReD.Rust**](https://github.com/synesissoftware/SyLVReD.Rust/);
* [**SyLVReDxx**](https://github.com/synesissoftware/SyLVReDxx/);


### License

**SyLVReD.Go** is released under the 3-clause BSD license. See [LICENSE](./LICENSE) for details.


<!-- ########################### end of file ########################### -->
