.Type()

Literals

NUMBER()
BOOL()
TEXT()
ERROR()
TYPE()
ANY()
TABLE(foo: TEXT)

TEXT or ERROR

FUNCTION(foo: TEXT) => TEXT

FUNCTION(s: TEXT) => BOOL or FUNCTION(s: NUMBER) => ERROR

- A new TYPE type
- every AST type has a new .Type() method that returns a []Variant

type BaseType string

type Variant struct {
    Base BaseType
    Properties map[string]BaseType
}

Example:
    {x: 4, y: ""}
    => TABLE(x: NUMBER, y: TEXT)

    x: some_func() NUMBER or TEXT
    {a: x, b: ""}
    => TABLE(a: NUMBER, y: TEXT) or
       TABLE(a: TEXT, y: TEXT)

    x(s):
        case type(s):
            text: true
            number: error("BRUH")
            bool: 1234
    => FUNCTION(s: TEXT) BOOL or
       FUNCTION(s: NUMBER) ERROR or
       FUNCTION(s: BOOL) NUMBER

- Every variant produces another variant
TABLE(x: NUMBER or TEXT) <= Not possible!
TABLE(x: NUMBER) or TABLE(x: TEXT) <= Possible!

# Type infering from other AST Nodes

## Type assertion
if a node type is an identifier, it can assert some type
Example:
    -a # asserts type(a) == NUMBER

Identifier: GetType from the Environment

NumberLiteral: NUMBER

BooleanLiteral: BOOL

TextLiteral: TEXT

PubStatement: GetType from the statement

CaseExpression:
    for every pattern, add a variant from the GetType from the pattern and block
    if subject is the result of Type.type(), asserts the type of the constant in the block.
    also asserts a potentional type of a variable from every pattern.

Block: GetType from the last ast node

PrefixExpression:
    if not: BOOL
    if -: NUMBER
    asserts the type of constant (not - BOOL, - - NUMBER)

InfixExpression:
+-*/ NUMBER (asserts both arguments are numbers), is, is not BOOL (asserts both arguments are the same type)

FunctionDeclaration:
    first, for every argument, it resolves all of the possible types and creates variants (including defaults)
    2. Resolves block
FunctionCall: The return type of FunctionDeclaration

TableLiteral:
    for every entry, resolves type and creates variants (also asserts the type from the entry value)

AccessOperator: gets type of subject, tries to get the property and gets it type (asserts type is Table or Text)

RestOperator: case by case basis

# Builtin function
Inference is impossible, so the types are filled out in the builtin lib.

# Error
x(y):
    Text.split(y) # y needs to be a text
    -y # y needs to be a number
    # y is required to be a TEXT and a NUMBER at the same time.

for every node, gets a constant assert and compares it.

FUNCTION(y: ANY)
1. line, asserts y is a TEXT and compares it to ANY (possible conversion)
FUNCTION(y: TEXT)
2. line, asserts y is a NUMBER and compares it to TEXT (impossible conversion, Error)

# Type inference example

using IO, Program


x(y):
    z: y
    rest: {exit_code: 17}
    Program.crash(z, ...rest)

x(1)

1. sets IO to TABLE(log: Builtin, get: Builtin) and Program to TABLE(crash: Builtin)
2. start of function declaration FUNCTION(y: ANY)
3. sets type of "z" to type(y) (ANY)
4. sets "rest" type to TABLE(exit_code: NUMBER)
5. Checks Program.crash contract, Program.crash exits