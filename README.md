# Smart Make Art (Smart) 🚀

[![Go Reference](https://pkg.go.dev/badge/github.com/extbit/smart.svg)](https://pkg.go.dev/github.com/extbit/smart)
[![Join the chat at https://gitter.im/duzy/smart](https://badges.gitter.im/duzy/smart.svg)](https://gitter.im/duzy/smart)

> **🚧 Status: Under Active Development (Beta)**  
> *Smart is currently mid-way through development. While the core JIT engine and syntax are functional, features and APIs are subject to change. We are actively seeking early adopters, feedback, and contributors to help shape the future of the project.*

**Smart Make Art** is a next-generation command-line utility and scripting language written in Go. Inspired by GNU `make`, Smart is explicitly designed to solve the pain points of compiling massive, hierarchical projects (like LLVM or Bitcoin Core) by offering a modular, data-typed, and multi-dialect build environment.

📚 [Read the Official Documentation](https://github.com/extbit/smart/wiki/Smart-Construction)

---

## Why Smart?

Building projects with complex hierarchies should be easy. 

While traditional `Makefile` relies on a single, easily-polluted global namespace, `smart` introduces strict modularity. Symbols and rules are safely contained within local module or project scopes, and dependencies are explicitly declared using `import` or `use` keywords.

### Key Features
* **True Modularity:** No global namespace clashes. Modules handle specific tasks and are `use`d by parent projects.
* **Data-Typed Macros:** Unlike GNU Make, Smart evaluates variables with native data types for safer, predictable scripting.
* **Multi-Dialect Recipes:** Write build recipes natively in `shell`, `python`, or generate files dynamically using `plain` text dialects.

---

## Quick Start

### 1. Install `smart`
Install the `smart` command-line utility directly via Go:

```shell
$ go install [github.com/extbit/smart/cmd@latest](https://github.com/extbit/smart/cmd@latest)
$ smart -help
