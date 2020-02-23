static int calc_rpn(const char *src)
{
    int stack[10];
    int sp = -1;

    for (int i = 0; src[i] != '\0'; i++) {
        if ('0' <= src[i] && src[i] <= '9') {  // digit
            stack[++sp] = src[i] - '0';
            continue;
        }

        switch (src[i]) {
        case '+':
            stack[sp - 1] = stack[sp - 1] + stack[sp];
            sp--;
            break;

        case '-':
            stack[sp - 1] = stack[sp - 1] - stack[sp];
            sp--;
            break;

        case '*':
            stack[sp - 1] = stack[sp - 1] * stack[sp];
            sp--;
            break;
        }
    }

    return stack[sp];
}

int main()
{
    return calc_rpn("3 4 + 2 1 - *");
}
