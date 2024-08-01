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
- add a default keyword for case [DONE]
- pattern matching with tables

{1} - Matches a table with one entry (O: 1)
{name: "frank"} - Matches a table with named entry (name: "frank")

{name: name}
if the name constant is defined, then it changes name to the value of name.
if its not, then it will bind the value of name: to it.

{1, ...} - Matches a table with one entry (0: 1) and variable number of other entries. if there is only one entry, panics.

{1, ...names}
if the names constant is defined, then it checks if names is a table, if it is, it compares it. if not it panics.
if it isnt, it assigns rest of table to names.

- add ... operator (the rest operator)
normally, it should do nothing.

in a function declaration, it means "put rest of the arguments in this table"
in a function call, it means "put all of the table entries as arguments to this function"
in a table literal, it means "add entries from this other table to this one"
also can be used in table literal as a "rest" matcher


- add variadic arguments
- named arguments support for builtin functions
- proper errors
- add some way of documentation

Error: Cannot reassign identifier x

```python
2. x: 4 # Problem
```
You already assigned x at
```python
1. x: 2
```

Why?
Zygon is immutable, blablabla
