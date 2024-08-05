# About Zygon

Zygon aims to be a simple functional language. The syntax is inspired by python.
The language is fully immutable. It will be completely type safe, yet without types.
It will also have a fully featured standard library.
Due to the semantics of the language (immutable, everything is pass by value) a garbage collector is not needed (matters when i get to compiling).

# Examples

## Hello World
```python
using IO

IO.log("Hello World")

```

## Fibonacci sequence

```python
using Program

fib(n):
    case:
        n < 0:  Program.crash("Incorrect number {n}")
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
