# gogen-dump

Generates low-level/high-performance binary (ie. non-textual but byte-stream) serialization/deserialization methods for Go `struct` type-defs.

## Usage:

at least:

    gogen-dump my/go/pkg/path

or better yet

    gogen-dump my/go/pkg/path myStruct OtherStruct

generates: `my/go/pkg/path/@serializers.gen.go`.

- Package path can be file-system (dir or file) path (rel/abs) or Go import path
- *If* the next arg is some `my-output-file-name.go`, then that one will be written instead of the default `@serializers.gen.go`.
- Optionally, all following args each name a struct type-def to be processed — if none, *all* declared in your package will be processed
  - recommended to explicitly state only the `struct` type-def names required, rather than let `gogen-dump` process any & all declared throughout your package.

## Satisfies my own following exact spec that I found underserved by the ecosystem at time of writing:

- no separate schema language / definition files: `struct` type-defs parsed from input `.go` source files serve as "schema" (so `gogen-dump` only generates methods, not types)
- no use of `reflect`, neither at generation time nor at at runtime, so private fields too can be (de)serialized
- unlike `gob` and most other (de)serialization schemes, does not (de)serialize field names (or even field/type IDs or tags, except for specially-tagged `interface{}`/`[]interface{}`-typed fields as described below) but rather purely follows (generation-time) type *structure*

### Compromises that make `gogen-dump` less-viable for *some* use-cases but still perfectly suitable for *others*:

- varints (`int`, `uint`, `uintptr`) always occupy 8 bytes regardless of native machine-word width
- caution: no support for / preservation of shared-references! pointees are currently (de)serialized in-place, no 'address registry' is kept
- caution: generated code imports and uses `unsafe` and thus assumes same endianness during serialization and deserialization — doesn't use `encoding/binary` or `reflect`
- caution: no schema/structural versioning or sanity/length checks

So by and large, use-cases are limited to local cache files of expensive-to-(re)compute (non-sharing) structures (but where the absence or corruption of such files at worst only delays but won't break the system), or IPC/RPC across processes/machines with identical endianness and where "schema version" will always be kept in sync (by means of architecture/ops discipline).

## Supports all built-in primitive types plus:

- other structs (or pointers/slices/maps/arrays/etc. referring to them) that have `gogen-dump`-generated un/marshaling, too
- any structs (or pointers/slices/maps/arrays/etc. referring to them) implementing both `encoding.BinaryMarshaler` and `encoding.BinaryUnmarshaler`
- `interface{}` (or `[]interface{}`) fields denoted as unions/sums via a *Go struct field tag* such as

        foo interface{} `gogen-dump:"bool []byte somePkg.otherType *orAnotherType"`

    (only concrete types should be named in there, no further interfaces, maximum of 255 entries)
- all of the above can be arbitrarily referred to in various nestings of pointers, slices, maps, arrays, pointers to pointers to slices of maps from arrays to pointers etc..
  - some truly unrealistic whacky combinations will generate broken/non-compiling code but these are really hard to find and typically —AFAICT so far— always something you'd never want in a proper code-base anyway (happy to be proven wrong!)
  - also multiple-indirection pointers (pointer-to-pointer or more levels) will be 'all-or-nothing' after a roundtrip, ie. if a later or the final pointer was `nil` at serialization time, the very foremost pointer will be `nil` after deserialization, in other words no preservation of occurrences of edge-cases like a-non-nil-pointer-to-(a-non-nil-pointer-to-.....)a-nil-pointer — another scenario that's unlikely to ever come up (as an irreconcilable issue) in any sane+clean real-world designs.
