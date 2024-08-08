.Type()

Literals

NUMBER
BOOL
STRING
ERROR
TYPE
ANY
TABLE(foo: STRING)

STRING || ERROR

FUNCTION(foo: STRING) => STRING

FUNCTION(s: STRING) => BOOL || FUNCTION(s: NUMBER) => ERROR

x(s):
    case type(s):
        string: true
        number: error("BRUH")