# gogen-dump

Generates high-performance binary (ie. non-textual but byte-stream) serialization/deserialization methods for Go `struct´ type-defs.

## Usage:

    gogen-dump my/go/pkg/path

### or:

    gogen-dump my/go/pkg/path myStruct OtherStruct

generates: `my/go/pkg/path/@serializers.gen.go`.

Package path can be file-system path or Go import path. If it is followed directly by `some-file-name.go`, then that one will be written instead of the default `@serializers.gen.go`.

> Very recommendable to explicitly state only the `struct` type names required, rather than let `gogen-dump` auto-grab-all declared in the package.

## Satisfies my own following exact spec that I found underserved by the ecosystem at time of writing:

- no separate schema language / definition files: `struct` type-defs parsed from input `.go` source files serve as "schema"
- no use of `reflect`, neither at generation time nor at at runtime, so private fields can be (de)serialized
- unlike `gob` and most other encoders, does not (de)serialize field names (or even field/type IDs), only goes by straight (generation-time) type *structure*
- varints (`int`, `uint`, `uintptr`) always occupy 8 bytes regardless of native machine-word width

Compromises —that make this non-viable for various use-cases but still perfectly suitable for various others— made for performance reasons:

- generated code imports and uses `unsafe` and thus assumes same endianness during serialization and deserialization — doesn't use `encoding/binary` or `reflect`
- no schema/structural versioning or sanity/length checks

So by and large, use-cases are limited to local cache files of expensive-to-(re)compute structures (but where the absence or corruption of such files at worst only slows down but won't break the system), or IPC/RPC across processes/machines with identical endianness and where "schema version" will always be in sync.

### Supports all built-in primitive types plus:

- other structs (or pointers/slices/maps/arrays/etc. referring to them) that have `gogen-dump`-generated un/marshaling, too
- any structs (or pointers/slices/maps/arrays/etc. referring to them) implementing both `encoding.BinaryMarshaler` and `encoding.BinaryUnmarshaler`
- `interface{}` (or `[]interface{}`) fields denoted as unions/sums via a *Go struct field tag* such as `gogen-dump:"bool []byte somePkg.otherType *orAnotherType"` (should be concrete types in there, no further interfaces, maximum of 255 entries)
- all these can be arbitrarily referred to in various nestings of pointers, slices, maps, arrays, pointers to pointers to slices of maps to pointers of arrays etc..
  - some truly unrealistic whacky combinations will generate broken/non-compiling code but these are really hard to find and typically always something you'd never want in a proper code-base anyway (AFAICT so far!)
  - also multiple-indirection pointers (pointer-to-pointer or more levels) will be 'all-or-nothing' after a roundtrip, ie. if a later or the final pointer was `nil` at serialization time, the very foremost pointer will be `nil` after deserialization, in other words no preservation of occurrences of edge-cases like a-non-nil-pointer-to-(a-non-nil-pointer-to-.....)a-nil-pointer — another scenario that's unlikely to ever come up (as an irreconcilable issue) in any sane+clean real-world designs.
