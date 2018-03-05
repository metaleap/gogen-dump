# gogen-dump

Generates high-performance binary (ie. byte-stream, non-textual) serialization/deserialization methods for Go `struct´ type-defs.

Fills the following underserved niche:

- `struct` type-defs parsed from input `.go` source files serve as "schema", no separate schema language/definition
- no use of `reflect`, neither at generation time nor at at runtime, so private fields can get (de)serialized
- does not write or read field names, only goes by straight type structure

Compromises (that make this non-viable for certain use-cases but still acceptably suitable for some others) made for performance reasons:

- uses `unsafe` (for speediest direct memory access to the actual multi-byte sequences holding individual field values) and thus assumes same endianness during serialization and deserialization — doesn't use `encoding/binary` or `reflect`
- no schema/structural versioning or sanity/length checks

So by and large, use-cases are limited to local cache files of expensive-to-(re)compute structures (but where the absence or corruption of such files at worst only slows down but won't break the system), or IPC/RPC across processes/machines with identical endianness and where "schema version" will always be in sync.

Supports all built-in primitive types plus:

- other structs (or pointers/slices/maps/arrays/etc. referring to them) that have `gogen-dump`-generated un/marshaling, too
- any structs (or pointers/slices/maps/arrays/etc. referring to them) implementing both `encoding.BinaryMarshaler` and `encoding.BinaryUnmarshaler`
- `interface{}` (or `[]interface{}`) fields denoted as unions/sums via a *Go struct field tag* such as `gogen-dump:"bool []byte somePkg.otherType *orAnotherType"` (should be concrete types in there, no further interfaces, maximum of 255 entries)
- all these can be arbitrarily referred to in various nestings of pointers, slices, maps, arrays, pointers to pointers to slices of maps to pointers of arrays etc.. — some truly unrealistic whacky combinations will generate broken/non-compiling code but these are truly hard to find and typically always something you'd never want in a proper code-base anyway, AFAICT so far
