# Contributing to Smart Make Art (Smart)

First off, thank you for considering contributing to Smart! 🎉

Smart is an ambitious project by ExtBit LLC aimed at superseding legacy build systems like `make` and `autotools` for massive, complex codebases. We are currently in **Beta** (mid-way through development), which means this is the perfect time to get involved. Your contributions—whether they are bug reports, feature requests, documentation improvements, or code—will actively shape the future of the engine.

The following is a set of guidelines for contributing to Smart. These are guidelines, not rules. Use your best judgment, and feel free to propose changes to this document in a pull request.

---

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. We expect all contributors to maintain a welcoming, inclusive, and harassment-free environment for everyone.

---

## How Can I Contribute?

### 1. Reporting Bugs

This section guides you through submitting a bug report. Following these guidelines helps maintainers and the community understand your report, reproduce the behavior, and find related bugs.

When reporting a bug, please include:
* **A clear and descriptive title.**
* **The exact version of Smart and Go** you are using.
* **Your operating system and environment.**
* **A minimal reproducible example:** If possible, include a minimal `do.smart` script that triggers the bug.
* **Steps to reproduce:** Explain exactly how to trigger the bug.
* **Expected vs. Actual behavior:** What did you expect to happen, and what actually happened? Include crash logs or stack traces if applicable.

### 2. Suggesting Enhancements

We are always looking for ways to make Smart faster, safer, and more intuitive. If you have an idea for a new feature or an enhancement to the JIT engine:

* **Check existing issues:** Your idea might already be under discussion.
* **Open a Feature Request:** Detail the problem your feature solves.
* **Provide an example:** Show how the proposed syntax or feature would look in a `do.smart` file.

### 3. Submitting Pull Requests

We love pull requests! To ensure a smooth review process, please follow these steps:

1. **Fork the repository** and create your branch from `main`.
2. **Discuss architectural changes first:** If you are planning a massive refactor or a significant change to the JIT/VM architecture, please open an issue first to discuss it with the core team.
3. **Write tests:** If you add code, add tests. If you are fixing a bug, add a test that catches the bug.
4. **Ensure the test suite passes:** Run `go test ./...` locally before submitting.
5. **Keep your commits clean:** Write clear, descriptive commit messages.
6. **Open the PR:** Describe exactly what your PR does, why it is needed, and link to any relevant issues.

---

## Development & Architecture Guidelines

Smart is not a standard Go web service; it is a high-performance, Stack Virtual Machine and JIT engine. Performance is a critical feature.

When writing code for the core engine, please keep the following in mind:

* **Zero-Allocation Philosophy:** The core tape evaluation (`symstr` and `vocab`) is designed to operate with mathematically zero heap allocations on the hot path. Avoid standard library functions that implicitly allocate (like `fmt.Sprintf` or `strings.Split`) in the core evaluation loop. Use the internal `Symbol` domain and compact builders instead.
* **Lexical Safety:** Smart relies on strict scope isolation. When adding new AST nodes or evaluation logic, ensure that lexical contexts do not leak.
* **Keep the VM Pure:** The `s.operands` stack should generally only hold primitive sizes, tokens, and opcodes. Avoid leaking arbitrary AST pointers onto the execution tape unless absolutely necessary for structural packing.

### Local Setup

To set up your local development environment:

```shell
# Clone your fork
git clone [https://github.com/YOUR_USERNAME/smart.git](https://github.com/YOUR_USERNAME/smart.git)
cd smart

# Download dependencies
go mod download

# Run the test suite
go test -v ./...

