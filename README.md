# About Zygon

Zygon aims to be a simple functional language. The syntax is inspired by python.
The language is fully immutable. It will be completely type safe, yet without types.
It will also have a fully featured standard library.
Due to the semantics of the language (immutable, everything is pass by value) a garbage collector is not needed (matters when I get to compiling).

# Examples

## Hello World
```python
using IO

IO.log("Hello World")

```

## Fibonacci sequence

```python
using Program

fib(n :: n > 0 or n is 0):
    case:
        n is 0: 0
        n is 1 or n is 2: 1
        default: fib(n-1) + fib(n-2)

fib(9) # Returns 34
```

## Web server setup (coming soon)
```python
using HTTP, HTML.(p, h1)

not_found():
    h1("404 Not Found")

greet(name):
    p("Hello, {name}!")

router(path):
    {"greet", name}: greet(name)
    default: not_found()

HTTP.serve(router, port: 8080)
```

# Style rules
- 4 spaced indentation
- modules are read as utf-8
- modules are named in pascal case
- functions and constants are named in snake case

# Ideas for the future
## Table change syntax
```python
{}
{_ | name: "Hello World"}
```
## Tracking site effects
```python
# The lang knows IO.log produces a side effect and can highlight it
do_stuff():
    IO.log("Hello World")

```

## Tracking dependencies


## Guards
```python
# We could write it like this
# Fun fact, the lang knows this could end the program and will highlight the side effect
fib(n):
    case n:
        n < 0: Program.end("incorrect input")
        0: 0
        n is 1 or n is 2: 1
        default: fib(n-1) + fib(n-2)

# But this will produce a compile time error, so its better
fib(n :: n > 0 or n is 0):
    case n:
        0: 0
        n is 1 or n is 2: 1
        default: fib(n-1) + fib(n-2)
        
# You could also call functions in the guard
create(admin :: is_admin(admin)): ...

# You can require a constant to be one of the defined states
file(mode :: Read or Write or Append): ...

# You can also reference parameters and spread them, for auto generation of states. The referenced constant will be evaluated at compile time.
# Represents select statement in SQL
select(db, table :: db.tables...): ...


```
## ...
Three dots behave like todo()! in Rust.
```python 
not_done(): ...
not_done() # Panics
```
## Docs and tests
Documentation comments will support markdown, references to parameters (@param) and everything will be checked at compile time. They also support things like TODO, FIX, etc. Also enforces SemVer. The syntax will probably change.
```python

docs {
    ver: "1.7.14"
    desc: "calculate the area of a circle given its @radius"
    examples: {
        circle_area(2) is 12.56636
    }
}
circle_area(radius :: radius > 0):
    Math.pi * Math.pow(radius, 2)

```

## Flow
```python
# if everything goes right, it returns the hash, otherwise, it returns the error()
flow:
    File.read(path): {contents: ...}
    Crypto.sha512(_.contents): {hash: ...}
    _.hash
```

## Before
Before executes code at compile time and inserts the result into the AST.
```python
x: before 4 * 10 - 20 # before turns it to 20 at compile time

# before also accepts a block
before:
    ...
```

## _
The _ variable is set to result of every evaluated expression, not just top level.
(I don't know about this yet)

## using statement wouldn't be needed
Everything would be auto imported
```python
IO.log("Hello World!")
My.Project.some_function()
```

## Applicable APIs will just be builder patterns
### RegEx
```python
using RegEx
pattern:
    RegEx.pattern()
    RegEx.text(_, "colo")
    RegEx.optional(_, "u")
    RegEx.text(_, "r")

RegEx.matches("color", pattern) # true
RegEx.matches("colour", pattern) # true
RegEx.matches("Hello World!", pattern) # false
```
### SQL
```python
using SQL
query:
    SQL.select(db, table: Articles, fields: All)

SQL.commit(query)
```
### HTML and CSS
Coming soon...
```python
```
## ?
The ? suffix can be used for printing debug info.
```python
x: 4
x? # Prints x: 4

y(z): z
y(x)? # Prints y(z: 4): 4
```