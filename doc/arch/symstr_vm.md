# Architecture Spec: Symbol-Stream Virtual Machine

## 1. Overview: The "Everything is a Symbol" Paradigm
The engine is transitioning from a traditional, deeply-nested Abstract Syntax Tree (AST) evaluator into a high-performance, JIT-compiled bytecode Virtual Machine. 

Instead of passing around heavy `Value` interfaces and constantly allocating new slices, **all data and computations are reduced to a contiguous stream of 32-bit integers (`Symbol`)**. 
* A `Symbol` acts as a universal assembly language. It can represent a character, a structural boundary, a cached number, or a complex Go object.
* Both Pattern Matching (structural/wildcards) and Evaluation (macros/builtins) share this exact same continuous medium.

---

## 2. Ephemeral Blobs: The Universal Pointer
To allow symbols to represent arbitrary objects without stringification, the `ephemeral` side-table is upgraded from `[]string` to `[]any` (a "blob" registry). This allows a `SymEph` integer to act as an `O(1)` pointer to complex states (e.g., `*statcache`, `*closure`, open sockets).

### 2.1 The GC Pinning Trap
When working with a long-lived `[]any` slice, strict lifecycle management is required. When an ephemeral symbol is freed, its slot in the array **must** be explicitly set to `nil`. If the slot is only pushed to the `freeEph` pool without zeroing, the global slice will retain a hidden reference to the object, preventing the Go Garbage Collector from ever freeing it.

### 2.2 The Zero-Allocation Boxing Rule
To prevent heap-escape allocations when binding objects to `any`, **raw structs must never be bound by value.** * ❌ `bindBlob(myStruct)` triggers an invisible heap allocation to box the struct.
* ✅ `bindBlob(&myStruct)` fits the pointer natively inside the interface with zero overhead.
    * See `internEphemeral` and `recycleEphemeral`
    * See `tempfile.cleanup`

---

## 3. Global vs. VM-Local Registries (Optimization)
Currently, `vocabulary` is a global structure protected by a `sync.RWMutex`.
* Immutable and deduplicated types (`strings`, `numbers`, `sequences`) safely remain in the global registry.
* **Ephemeral Blobs** should ideally migrate to a **VM-Local execution state** (e.g., directly inside `symstr` or `Context`).
  * **Benefit 1:** Lock-free, `O(1)` read/writes during execution.
  * **Benefit 2:** When the VM halts, the local array is inherently dropped, instantly GC'ing all intermediate ephemeral objects without requiring manual `freeBlob` tracking.

---

## 4. The Execution Flow

The engine operates in a continuous, lossless loop between AST representation and execution bytecode.

### Phase 1: JIT Compilation (`opEvalValue`)
The VM consumes an AST node (`Value`).
* **Passive Structural Nodes** (e.g., `path`, `compound`, `list`) are destructured into raw symbols and structural punctuation (e.g., `symSlash`, `symDot`).
* **Active Dynamic Nodes** (e.g., `auto`, `delegate`, `arrow`, `builtin`) are **Expanded / Fused** immediately into concrete data or function calls before the VM attempts to match them.

### Phase 2: Execution (`symstr.step / exhaust`)
The VM executes the opcodes using a **Stack-Based Calling Convention**.
* Rather than scanning the symbol stream for `symLparen` and `symRparen`, builtins rely on arguments that have been fully evaluated and pushed onto `s.vmstack`.
* `opEvokeBuiltin` pops `N` items off the stack, executes the Go routine, and pushes/emits the resulting symbol(s) to the ledger (`vmtape`).
  * `opEvoke*` should not reuse AST, it underlay values of `string`, `numbers`, and `blob`.

### Phase 3: Repacking (`__symPackValue`)
The final ledger is a flat array of `uint32` symbols (e.g., `[symFoo, symSlash, symBar, symDot, symO]`). The `__symPackValue` engine parses this contiguous slice back into a concrete AST tree.
* **Adjacency Hierarchy:** Symbols are aggregated hierarchically. Adjacent symbols form `compound`s, dot-separated components form `qualword`s, and slash-separated components form `path`s.
* **Lossless Boundaries:** Macros expanding to lists must emit grouping tokens (e.g., `symLtopcorner`...`symRbotcorner`) so the repacker can accurately restore the AST bounds.

---

## 5. Upcoming Implementation Milestones

1. **Vocabulary Upgrade:** * Modify `vocabulary.ephemeral` to `[]any`.
   * Implement GC-safe `bindBlob` and `freeBlob` (or migrate to VM-local blob arrays).
     * See `internEphemeral` and `recycleEphemeral`
     * See `tempfile.cleanup`
2. **Stack-Based Execution:** * Transition `opEvokeBuiltin` to consume from `s.vmstack` instead of parsing the stream.
3. **Punctuation Fidelity:** * Audit `opEvalValue` to ensure all composite types inject the correct delimiters (`symSpace`, boundaries) into the stream so `__symPackValue` can seamlessly unpack them.
4. **JIT Expansion Strategy:** * Ensure `opEvalValue` eagerly expands `*auto`, `*delegate`, and `*builtin` before dropping into literal matching.
