static int pc = 0;
static const char *text;

static int var[128];

#define NEXT (text[pc++])
#define PEEK (text[pc])

int next(void)
{
    while (text[pc] == ' ') pc++;
    return text[pc++];
}

int peek(void)
{
    while (text[pc] == ' ') pc++;
    return text[pc];
}

int interpret(void)
{
    char ch = next();

    if (ch == '(') {
        ch = next();

        switch (ch) {
        default:
            return -1;  // error

        case '+': {
            int val = interpret();
            val += interpret();
            next();  // Eat ')'.
            return val;
        }

        case '-': {
            int val = interpret();
            val -= interpret();
            next();  // Eat ')'.
            return val;
        }

        case 'l': {
            next();  // Eat 'e'.
            next();  // Eat 't'.
            next();  // Eat '('.
            next();  // Eat '('.
            char id = next();
            int val = interpret();
            var[(int)id] = val;
            next();  // Eat ')'.
            next();  // Eat ')'.
            return interpret();
        }
        }
    }

    if ('0' <= ch && ch <= '9') return ch - '0';

    return var[(int)ch];
}

int lisp(const char *str)
{
    pc = 0;
    text = str;

    int val = interpret();
    return val;
}

int main()
{
    return lisp(
        "(let ((a (+ (+ (+ 9 9) (+ 9 9)) 9)))"
        "   (- a 3))");
}
