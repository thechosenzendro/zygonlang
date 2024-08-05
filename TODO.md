- change accessor syntax [DONE]
- change modules to tables [DONE]
- make the code robust
- add the static analyzer (the whole reason this project started in the first place)
- add a way to change a table (Table.change) [DONE]
- add IO.get (input) [DONE]
- make OR short circuiting [DONE]
- add Program.crash (panic) [DONE]
- decide on the style rules of the language [DONE]
- add error value [DONE]
- organize the code better
- allow use of user defined modules (search first at project root than lib root) [DONE]
- fix calling functions in case patterns [DONE]
- add a default keyword for case [DONE]
- a file can import itself, and there can be an import loop - resolve
- pattern matching with tables [DONE]

{1} - Matches a table with one entry (O: 1) [DONE]
{name: "frank"} - Matches a table with named entry (name: "frank") [DONE]

{name: name}
this will bind the value of name: to the name constant. [DONE]

{1, ...} - Matches a table with one entry (0: 1) and variable number of other entries. if there is only one entry, it fails. [DONE]

{1, ...names}
assignes rest of the table to "names" [DONE]
- add ... operator (the rest operator) [DONE]
normally, it should do nothing.

in a function declaration, it means "put rest of the arguments in this table" [DONE]
in a function call, it means "put all of the table entries as arguments to this function" [DONE]
in a table literal, it means "add entries from this other table to this one" [DONE]
also can be used in table literal as a "rest" matcher [DONE]


- add variadic arguments [DONE]
- named arguments support for builtin functions [DONE]
- add contracts to builtin functions [DONE]
- add builtin values [DONE]
- using should set the last identifier in path instead of the first [DONE]
- blocks should have its  own env
- proper errors
- add some way of documentation
- add package management
