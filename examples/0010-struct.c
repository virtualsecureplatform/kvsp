struct hoge {
    long long int x, y;
};

struct hoge piyo(struct hoge h)
{
    if (h.x < 0) return h;

    h.x -= 1;
    h.y -= 1;
    struct hoge ret = piyo(h);
    return ret;
}

int main()
{
    struct hoge h = {1, 3};
    return piyo(h).y;
}
