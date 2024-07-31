- change accessor syntax [DONE]
- change modules to tables [DONE]
- make the code robust
- add the static analyzer (the whole reason this project started in the first place)
- add a way to change a table [DONE]
- add io.get (input) [DONE]
- make AND and OR short circuiting
- add program.crash (panic) [DONE]
- decide on the style rules of the language
- add error value
- organize the code better
- pattern matching with tables
- add ... operator
- add variadic arguments
- named arguments support for builtin functions
- proper errors

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
