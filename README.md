# About Zygon

Zygon aims to be a simple functional language. The syntax is inspired by python.
The language is fully immutable. It will be completely type safe, yet without types.
It will also have a fully featured standard library.

# Examples

## Hello World
```python
using io

io.log("Hello World")

```

## Fibonacci sequence

```python
using io

fib(n):
    case:
        n < 0:  io.fatal("Incorrect number {n}")
        n is 0: 0
        n is 1 or n is 2: 1
        true: fib(n-1) + fib(n-2)

fib(9) # Returns 34
```