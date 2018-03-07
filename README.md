# gogen-dump

Generates **low-level/high-performance binary serialization**/deserialization methods for the Go `struct` type-defs you already have:

### *no schema files, practically no restrictions, no safety hatches*!

## Usage:

at least:

    gogen-dump my/go/pkg/path

or better yet

    gogen-dump my/go/pkg/path myStruct OtherStruct

generates: `my/go/pkg/path/@serializers.gen.go`.

- **Package path** can be file-system (dir or file) path (rel/abs) or Go import path
- *If* the very next arg is some `my-output-file-name.go`, then that one will be written instead of the default `@serializers.gen.go`.
- Optionally, all the following args each name a struct type-def to be processed (recommended) — if none, *all* declared in your package will be processed (undesirable)
- (Some more exotic flags are offered, see bottom of this doc)

## Satisfies my own following exact spec that I found underserved by the ecosystem at time of writing:

- no separate schema language / definition files: `struct` type-defs parsed from input `.go` source files serve as "schema" (so `gogen-dump` only generates methods, not types)
- thanks to the above, no use of `reflect`-based struct type-def introspection at runtime or code-gen time, so private fields too can be (de)serialized
- unlike `gob` and most other (de)serialization schemes, does not de/encode field names (or even field/type IDs or tags, except for specially-tagged interface-typed fields as described below) but rather purely follows (generation-time) type *structure*

### Compromises that make `gogen-dump` less-viable for *some* use-cases but still perfectly suitable for *others*:

- varints (`int`, `uint`, `uintptr`) always occupy 8 bytes regardless of native machine-word width
- caution: no support for / preservation of shared-references! pointees are currently (de)serialized in-place, no 'address registry' is kept
- caution: generated code imports and uses `unsafe` and thus assumes same endianness during serialization and deserialization — doesn't use `encoding/binary` or `reflect`
- caution: no explicit or gradual versioning or sanity/length checks

So by and large, use-cases are limited to scenarios such as:
- local cache files of expensive-to-(re)compute (non-sharing) structures (but where the absence or corruption of such files at worst only delays but won't break the system),
- or IPC/RPC across processes/machines with identical endianness and where "schema" structure will always be kept in sync (by means of architecture/ops discipline)
- any-and-all designs where endianness and `struct`ural (not nominal) type identities are guaranteed to remain equivalent between the serializing and deserializing parties and moments-in-time (or where a fallback mechanism is sensible and in place).

## Supports all built-in primitive-type fields plus:

- fields to other structs (or pointers/slices/maps/arrays/etc. referring to them) that have `gogen-dump`-generated (de)serialization, too
- fields to any types (or pointers/slices/maps/arrays/etc. referring to them) implementing *both* `encoding.BinaryMarshaler` and `encoding.BinaryUnmarshaler`
- interface-typed fields denoted as unions/sums via a *Go struct field tag* such as

        myField interface{} `gogen-dump:"bool []byte somePkg.otherType *orAnotherType"`

    (only concrete types should be named in there, no further interfaces, maximum of 255 entries, also works for slice/array/pointer/map(value)-of-interface-type fields/values)
- fields to directly placed (but not indirectly-referenced) inline in-struct 'anonymous' sub-structs to arbitrary nesting depths
- all of a `struct`'s embeds are 'fields' (and dealt with as described above+below), too, for our purposes (as indeed internally they are anyway)
- all of the above (except inline in-struct 'anonymous' sub-structs) can be arbitrarily referred to in various nestings of pointers, slices, maps, arrays, pointers to pointers to slices of maps from arrays to pointers etc..
  - some truly unrealistic whacky combinations will generate broken/non-compiling code but these are really hard to find and typically —AFAICT so far— always something you'd never want in a proper code-base anyway (happy to be proven wrong!)
  - also multiple-indirection pointers (pointer-to-pointer or more levels) will be 'all-or-nothing' after a roundtrip, ie. if a later or the final pointer was `nil` at serialization time, the very foremost pointer will be `nil` after deserialization, in other words no preservation of occurrences of edge-cases like a-non-nil-pointer-to-(a-non-nil-pointer-to-.....)a-nil-pointer — another scenario that's unlikely to ever come up (as an irreconcilable issue) in any sane+clean real-world designs.

## Further optional flags for tweakers:

- `-safeVarints` — if present, all varints (`int`, `uint`, `uintptr`) are explicitly type-converted from/to `uint64`/`int64` during `unsafe.Pointer` shenanigans at serialization/deserialization time. (**If missing** (the default), varints are *also still* always written-to and read-from 8-byte segments during both serialization and deserialization —both in the source/destination byte-stream and local source/destination memory—, but without any such explicit type conversions.)
- `-optVarintsInFixedSizeds` — faster code is generated for fixed-size fields (incl. structs and arrays that themselves contain no slices, maps, pointers, strings), but varints (`int`, `uint`, `uintptr`) are not considered "fixed-size" for this purpose by default. **If this flag is present**, they will be and then wherever this faster serialization logic occurs, the points made above (for `-safeVarints`) no longer apply and varints occupy the number of bytes dictated by the current machine-word width (4 bytes on 32-bit, 8 bytes on 64-bit), meaning source and destination machines must match not just in endianness and "schema" structure but also in that respect.
- `-ignoreUnknownTypeCases` — if present, serialization of interface-typed fields with non-`nil` values of types not mentioned in its tagged-union field-tag (see previous section) simply writes a type-tag byte of `0` (equivalent to value `nil`) and subsequent deserialization will restore the field as `nil`. **If missing** (the default), serialization raises an error as a sanity check reminding you to update the tagged-union field-tag.
