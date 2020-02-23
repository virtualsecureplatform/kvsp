// Thanks to: http://nyanp.hatenablog.com/entry/20110728/p1
#define $ ((((((((0
#define X  <<1|1)
#define _  <<1)//))))))

typedef unsigned char uint8_t;

enum {
    SIZE = 8,
};

static uint8_t at(uint8_t map[], int x, int y)
{
    return (map[y] >> (7 - x)) & 1;
}

/*
static uint8_t safe_at(uint8_t map[], int x, int y)
{
    if (x < 0) x += 8;
    if (x >= 8) x -= 8;
    if (y < 0) y += SIZE;
    if (y >= SIZE) y -= SIZE;

    return at(map, x, y);
}
*/

static void step(uint8_t map[])
{
    uint8_t src[SIZE];
    for (int i = 0; i < SIZE; i++) src[i] = map[i];

    for (int y = 1; y < SIZE - 1; y++) {
        int res = 0;

        for (int x = 1; x < 8 - 1; x++) {
            uint8_t p = at(src, x, y);

            int cnt = 0;

            if (at(src, x - 1, y - 1)) cnt++;
            if (at(src, x, y - 1)) cnt++;
            if (at(src, x + 1, y - 1)) cnt++;

            if (at(src, x - 1, y)) cnt++;
            if (at(src, x + 1, y)) cnt++;

            if (at(src, x - 1, y + 1)) cnt++;
            if (at(src, x, y + 1)) cnt++;
            if (at(src, x + 1, y + 1)) cnt++;

            if (!p && cnt == 3) res |= (1 << (7 - x));
            if (p && (cnt == 2 || cnt == 3)) res |= (1 << (7 - x));
        }

        map[y] = res;
    }
}

int main()
{
    static uint8_t map[SIZE] = {
        $ _ _ _ _ _ _ _ _,  //
        $ _ _ X _ _ _ _ _,  //
        $ _ _ _ X _ _ _ _,  //
        $ _ X X X _ _ _ _,  //
        $ _ _ _ _ _ _ _ _,  //
        $ _ _ _ _ _ _ _ _,  //
        $ _ _ _ _ _ _ _ _,  //
        $ _ _ _ _ _ _ _ _,  //
    };

    step(map);
    step(map);
    step(map);
    step(map);

    return ((map[1] >> 3) << 12) | ((map[2] >> 3) << 8) | ((map[3] >> 3) << 4) |
           (map[4] >> 3);
}
