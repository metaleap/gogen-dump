# gogen-dump

Generates **low-level/high-performance binary serialization**/deserialization methods for the Go `struct` type-defs you already have:

### *no schema files, no `reflect`; profoundly few restrictions, safety hatches optional*!

## Usage:

at least:

    gogen-dump my/go/pkg/path

or better yet

    gogen-dump my/go/pkg/path myStruct OtherStruct

generates: `my/go/pkg/path/zerealizers.gen.go`.

- **Package path** can be file-system (dir or file) path (rel/abs) or Go import path
- *If* the very next arg is some `my-output-file-name.go`, then that one will be written instead of the default `zerealizers.gen.go`.
- Optionally, all the following args each name a struct type-def to be processed (recommended) — if none, *all* declared in your package will be processed (undesirable)
- (Some more exotic flags to be appended last are also offered, see bottom of this doc)

For each (selected) `struct` that has any serializable fields at all, the following methods are generated:

```go
    // internal serialization code, only pure raw data, no meta-headers
    marshalTo(bytes.Buffer) (error)

    // internal code to deserialize marshalTo's output, no sanity checks
    unmarshalFrom(*int, []byte) (error)

    // implements encoding.BinaryMarshaler using marshalTo
    MarshalBinary() ([]byte, error)

    // implements encoding.BinaryUnmarshaler using unmarshalFrom
    UnmarshalBinary([]byte) error

    // implements io.WriterTo, writes 16B sig+len header and then marshalTo's output
    WriteTo(io.Writer) (int64, error)

    // implements io.ReaderFrom, verifies the 16B header, calls unmarshalFrom
    ReadFrom(io.Reader) (int64, error)
```

## Satisfies my own following fuzzy in-flux spec that I found underserved by the ecosystem at time of writing:


- no separate schema language / definition files: `struct` type-defs parsed from input `.go` source files serve as "schema" (so `gogen-dump` only generates methods, not types)
- thanks to the above, no use of `reflect`-based introspection, so private fields too can be (de)serialized
- unlike `gob` and most other (de)serialization schemes, does not laboriously encode and decode field or type names/tags/IDs (except as desired for specially-tagged interface-typed fields, detailed below) but rather purely follows (generation-time) type *structure*: the code is the schema, the byte stream is pure raw data — not always what you want, but quite often what I require
- generates reads and writes that pack the most bytes-transferred in the fewest instructions, so attempts to have the largest-feasible contiguous-in-memory pointable-to data chunk (aka. statically known fixed-size field/structure/array/combination) done in ideally just a single memory copy (or as few as necessary)

### Compromises that make `gogen-dump` less-viable for *some* use-cases but still perfectly suitable for *others*:

- inherently not as massively scalable as the *"schema-generated struct-lookalike accessor-methods over underlying incoming/outgoing byte-streams"* approach taken by eg. CapnProto, FlatBuffers and their ilk — but OTOH we get better developer ergonomics working with our native-Go structs first-class as we're used to, transforming / mangling / accumulating / allocating / destroying / (de)referencing them freely as-usual and in-memory in the (potentially) long time between de/serializations.
- varints (`int`, `uint`, `uintptr`) always occupy 8 bytes regardless of native machine-word width (except in fixed-size fields/structures (unless `--varintsNotFixedSize` on), described further below)
- caution: no support for / preservation of shared-references! pointees are currently (de)serialized in-place, no "address registry" for storage and restoral of sharing is kept
  - this isn't likely to change any time soon unless I run into a compelling need for this: for the same scenarios where you'd want low-level/high-perf (de)serialization (vs. the std-lib built-ins) *in the first place*, you'd likely also lay out the participating data structures tightly-packed without (m)any (de)references/invariants and for the few truly required sharing needs occurring, you'd in place of pointers rely on your own old-school 'addressing' = indexing (slices/maps) — which will serialize neatly.
- caution: generated code uses `unsafe.Pointer` everywhere extensively and thus assumes same endianness during serialization and deserialization — doesn't use `encoding/binary` or `reflect`

So by and large, use-cases are limited to scenarios such as:
- local cache files of expensive-to-(re)compute (non-sharing) structures (but where the absence or "corruption" —aka. "schema"-`struct`ural "version" changes— of such files at worst only delays but won't break the system),
- or IPC/RPC across processes/machines with identical endianness and where "schema" structure will always be kept in sync (by means of architectural/ops practice/discipline)
- more generally, any-and-all designs where endianness and `struct`ural (not nominal) type identities are guaranteed to remain equivalent between the serializing and deserializing parties and moments-in-time (or where a fallback mechanism is sensible and in place).

## Supports all `builtin` primitive-type fields plus:

- fields to other in-package `struct`s (or pointers/slices/maps/arrays/etc. referring to them) that have `gogen-dump`-generated (de)serialization, too
- fields to any (non-interface) types (or pointers/slices/maps/arrays/etc. referring to them) implementing *both* `encoding.BinaryMarshaler` and `encoding.BinaryUnmarshaler`
  - all imported, ie. package-external types (excepting those aliased via `ggd:"foo"` struct-field-tag or `--bar.Baz=foo` flag described further down) will need to implement these two; all `gogen-dump`-generated source files also furnish these implementations
- fields to in-package type synomyns and type aliases handled as described above+below
- interface-typed fields denoted as unions/sums via a *Go struct-field tag* such as

        myField my.Iface `ggd:"bool []byte somePkg.otherType *orAnotherType"`

    (only concrete types should be named in there: no further interfaces; minimum of 2 and maximum of 255 entries; also works equivalently for slice/array/pointer/map(value)-of-interface-type fields/values or same-package type aliases/synonyms of such)
- fields to directly included (but not indirected via pointer/slice/etc.) "inline" in-struct "anonymous" sub-structs to arbitrary nesting depths
- all of a `struct`'s *embeds* are "fields", too (and dealt with as described above+below) for our purposes here
- all of the above (except "inline" in-struct "anonymous" sub-structs) can be arbitrarily referred to in various nestings of pointers, slices, maps, arrays, pointers to pointers to slices of maps from arrays to pointers etc..

## Further optional flags to append for tuners and tweakers:

- all flags can be given via `--` or `-`.

- `--stdlibBytesBuffer`
  - if missing (the default), `gogen-dump` generates a package-local `bytes.Buffer`-lookalike (just much more minimalist as its meant to buffer only writes, not reads) named `writeBuf`
  - if present, this will not be generated and the standard library's `bytes.Buffer` is used during marshalings.

- `--safeVarints`
  - if present, all varints (`int`, `uint`, `uintptr`) are explicitly type-converted from/to `uint64`/`int64` during `unsafe.Pointer` shenanigans at serialization/deserialization time.
  - if missing (the default), varints are *also still* always written-to and read-from 8-byte segments during both serialization and deserialization —both in the source/destination byte-stream and local source/destination memory—, but without any such explicit type conversions.

- `--noFixedSizeCode`
  - if missing (the default), `gogen-dump` generates much terser+faster code-paths for (de)serializing chunks of contiguous fixed-size data in fewer instructions. (This necessitates a 100% equivalence between the marshaling and unmarshaling Go binaries' compiler/architecture with regards to machine-word size and struct padding/alignment/offset — not just endianness.)
  - if present, no such code is ever generated, so even fixed-length arrays (of fixed-size elements) are (de)serialized by iteration and fixed-size structs field-by-field etc. On the flip side, both parties only need to share the same endianness (and de-facto "schema version").

- `--varintsNotFixedSize`
  - when much terser+faster code is generated as mentioned above for known-to-be-fixed-size fields / field sequences (incl. structs and statically-sized-arrays that themselves contain no slices, maps, pointers, strings), varints (`int`, `uint`, `uintptr`) too are considered "fixed-size" for this purpose by default (though that 'fixed' size varies depending on the compiler/arch combination used)
  - if this flag is present, they *aren't*, meaning that fixed-size arrays of varints or otherwise-fixed-size structs with varints are no longer considered for generating fixed-size-optimized code-paths

- `--ignoreUnknownTypeCases`
  - if present, serialization of interface-typed fields with non-`nil` values of types not mentioned in its tagged-union field-tag (see previous section) simply writes a type-tag byte of `0` (equivalent to value `nil`) and subsequent deserialization will restore the field as `nil`
  - if missing (the default), serialization raises an error as a sanity check reminding you to update the tagged-union field-tag.

- `--sort.StringSlice=[]string`, `--sql.IsolationLevel=int`, `--os.FileMode=uint32`, etc.
  - declares as a type synonym/alias the specified type used in but not defined in the current package, to generate low-level (de)serialization code for fields/elements of such package-external types that are really just aliases for (direct or indirected) prim-types (and often do not implement `encoding.BinaryMarshaler` / `encoding.BinaryUnmarshaler`).
  - For convenience, `--time.Duration=int64` is already always implicitly present and does not need to be expressly specified.
  - Reminder that in-package type aliases / synonyms will be picked up automatically and need not be expressly specified.
  - An alternative to type-aliasing via command-line flags is a `ggd:"underlyingtypename"` struct-field-tag next to the "offender" in your source `struct`(s). It only needs to exist once, not for every single applicable field.
