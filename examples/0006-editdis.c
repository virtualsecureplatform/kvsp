static int min3(int a, int b, int c)
{
    if (a < b) {
        if (b < c) return a;
        if (c < a) return c;
        // a < b && a <= c
        return a;
    }

    // b <= a
    if (b < c) return b;
    // b <= a && c <= b
    return c;
}

static int strlen(const char *s)
{
    int i = 0;
    while (s[i] != '\0') i++;
    return i;
}

static void swap_intp(int **lhs, int **rhs)
{
    int *tmp = *lhs;
    *lhs = *rhs;
    *rhs = tmp;
}

static int calc_edit_distance(const char *str0, const char *str1)
{
    static int dp[2][7 + 1];

    int *dp0 = dp[0], *dp1 = dp[1];
    int n = strlen(str0), m = strlen(str1);

    // Initialization of array `dp`.
    for (int i = 0; i <= n; i++) dp0[i] = i;
    dp1[0] = 0;

    // Let's DP.
    for (int i = 1; i <= n; i++) {
        for (int j = 1; j <= m; j++) {
            dp1[j] = min3(dp0[j] + 1, dp1[j - 1] + 1,
                          dp0[j - 1] + (str0[i] == str1[i] ? 0 : 1));
        }
        swap_intp(&dp0, &dp1);
    }

    return dp0[m];
}

int main()
{
    return calc_edit_distance("kitten", "sitting");
}
