- change accessor syntax [DONE]
- change modules to tables [DONE]
- make the code robust [DONE]
- add the static analyzer (the whole reason this project started in the first place)
- add a way to change a table (Table.change) [DONE]
- add IO.get (input) [DONE]
- make OR short circuiting [DONE]
- add Program.crash (panic) [DONE]
- decide on the style rules of the language [DONE]
- add error value [DONE]
- organize the code better [DONE]
- considering giving the access operator free will
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
identifier _ means any value

- add variadic arguments [DONE]
- named arguments support for builtin functions [DONE]
- add contracts to builtin functions [DONE]
- add builtin values [DONE]
- using should set the last identifier in path instead of the first [DONE]
- blocks should have its  own env [DONE]
- proper errors

   _____Main.zygon_____
1 │ using HTTP.
               ^ Expected ( after this

   _____Main.zygon_____
1 │ IO.log(name)
           ^^^^ "name" is not defined

   _____Main.zygon_____
1 │ IO..log(name)
       ^ there shouldnt be a colon there

   _____Main.zygon_____
1 │ "Hello {name}!
                  ^ unterminated string

   _____Main.zygon_____
1 │ index(ctx)
          ^^^ "ctx" does not have the "path" attribute

- add some way of documentation
- add package management
- add rule that every case needs a default
- enforce rules (using and pub only at the top, named arguments after positional, rest after everything)
using and pub only at the top [DONE]
function call named args after positional [DONE]
function call rest after everything [DONE]
function declaration rest after everything [DONE]
- add alternative patterns for pattern matching
- add chain construct
chain:
	DB.get(db, ctx.session): {id: _, ...}
	User.is_admin(_): true

for every line, it evals the expression, pattern matches the result,
	if it matches, it sets the _ variable to the result and moves on
	if it does not, it returns the result

- add static typing
assert(BaseType, nil) asserts only BaseType
assert(BaseType, map[]...) asserts everything
Any type when type can be anything
assert on tables does not assert the exact shape, but it ensures that every entry is present

{x: 4, y: "Hello World"} can be of type Table{x: Number}

ResolveType() returns
	Identifier: gets type from the enviroment
	NumberLiteral: Number
	BooleanLiteral: Bool
	TextLiteral: Text
	PubStatement: nil
	CaseExpression:
		accumulates every entry's ResolveType() and returns a union
	Block: ResolveType()'s every line and returns the last type
	PrefixExpression:
		NOT: Bool
		MINUS: Number
	InfixExpression:
		+ - * /: Number
		is, is not: Bool
	FunctionDeclaration:
		for every argument if it has a default, it ResolveType()'s it, else it's nil
		ResolveType()'s body
		for every nil in argument, it gets type from the body's env
	FunctionCall:
		returns the return of its FunctionDeclaration
	TableLiteral:
		for every entry, ResolveType()
	AccessOperator: gets type of entry
	RestOperator: figure out

AST Nodes assert types of their arguments 
	PrefixExpression:
		NOT: asserts argument is bool
		MINUS: asserts argument is number
	InfixExpression:
		+ - * /: asserts both arguments are numbers
		is, is not: asserts both arguments are of the same typee
	FunctionCall:
		asserts Fn is a Function
		for every argument, it gets its type and asserts that parameter is of that type 
	AccessOperator:
		asserts argument is Table or Text
		asserts argument has attribute

	RestOperator: asserts argument is table


Unions
How should the type system represent a value being multiple types?

negate(x):
	case Type.type(x):
		Type.number: -x
		Type.bool: not x

The type system currently represents it like this:
Function{x: Number or Bool} Number or Bool

But thats not right!
This type implicitly says that x COULD BE Number and return type COULD BE BOOL.
That just does not make sense.

Better representation is this:
Union{Function{x: Number} Number, Function{x: Bool}: bool}
This is much better!
But how do we do this?
IDK. Will figure it out later.