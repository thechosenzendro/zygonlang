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
        true: fib(n-1) + fib(n-2)

fib(9) # Returns 34
```

# Style rules
- 4 spaced indentation
- modules are read as utf-8
- modules are named in pascal case
- functions and constants are named in snake case
