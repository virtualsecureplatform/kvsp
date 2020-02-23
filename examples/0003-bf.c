static void jumpFront(const char *str, int *index)
{
    if (str[*index] != '[') return;
    for (int count = 0; str[*index] != '\0'; (*index)++) {
        if (str[*index] == '[')
            count++;
        else if (str[*index] == ']')
            count--;
        if (count == 0) break;
    }
}

static void jumpBack(const char *str, int *index)
{
    if (str[*index] != ']') return;
    for (int count = 0; *index >= 0; (*index)--) {
        if (str[*index] == '[')
            count--;
        else if (str[*index] == ']')
            count++;
        if (count == 0) break;
    }
}

static int brainfuck(const char *str)
{
    static int data[100];

    int pointer = sizeof(data) / sizeof(int) / 2;
    for (int index = 0; str[index] != '\0'; index++) {
        switch (str[index]) {
        case '+':
            data[pointer]++;
            break;
        case '-':
            data[pointer]--;
            break;
        case '>':
            pointer++;
            break;
        case '<':
            pointer--;
            break;
        case '.':
            break;
        case ',':
            break;
        case '[':
            if (data[pointer] == 0) jumpFront(str, &index);
            break;
        case ']':
            if (data[pointer] != 0) jumpBack(str, &index);
            break;
        }
    }
    return data[pointer];
}

int main()
{
    return brainfuck("++++[>++++++++++<-]>++");
}
