#include <stddef.h>
#include <stdint.h>

void *memcpy(void *dst, const void *src, size_t n)
{
    uint8_t *d = (uint8_t *)dst;
    const uint8_t *s = (const uint8_t *)src;
    for (size_t i = 0; i < n; i++)
        d[i] = s[i];
    return dst;
}

void *memmove(void *dst, const void *src, size_t n)
{
    uint8_t *d = (uint8_t *)dst;
    const uint8_t *s = (const uint8_t *)src;
    if (d < s) {
        for (size_t i = 0; i < n; i++)
            d[i] = s[i];
    }
    else {
        for (size_t i = n; i > 0; i--)
            d[i - 1] = s[i - 1];
    }
    return dst;
}

void *memset(void *dst, int c, size_t n)
{
    uint8_t *d = (uint8_t *)dst;
    for (size_t i = 0; i < n; i++)
        d[i] = (uint8_t)c;
    return dst;
}

int memcmp(const void *lhs, const void *rhs, size_t n)
{
    const uint8_t *l = (const uint8_t *)lhs;
    const uint8_t *r = (const uint8_t *)rhs;
    for (size_t i = 0; i < n; i++) {
        if (l[i] != r[i])
            return (int)l[i] - (int)r[i];
    }
    return 0;
}

size_t strlen(const char *s)
{
    size_t n = 0;
    while (s[n] != '\0')
        n++;
    return n;
}

int __mulsi3(int a, int b)
{
    uint32_t x = (uint32_t)a;
    uint32_t y = (uint32_t)b;
    uint32_t ret = 0;
    while (y != 0) {
        if ((y & 1u) != 0)
            ret += x;
        x <<= 1;
        y >>= 1;
    }
    return (int)ret;
}

static uint32_t udivmodsi4(uint32_t n, uint32_t d, uint32_t *rem)
{
    uint32_t q = 0;
    uint32_t r = 0;

    if (d == 0) {
        if (rem)
            *rem = n;
        return UINT32_MAX;
    }

    for (int i = 31; i >= 0; i--) {
        r = (r << 1) | ((n >> i) & 1u);
        if (r >= d) {
            r -= d;
            q |= 1u << i;
        }
    }

    if (rem)
        *rem = r;
    return q;
}

unsigned __udivsi3(unsigned n, unsigned d)
{
    return udivmodsi4(n, d, NULL);
}

unsigned __umodsi3(unsigned n, unsigned d)
{
    uint32_t r = 0;
    udivmodsi4(n, d, &r);
    return r;
}

int __divsi3(int n, int d)
{
    uint32_t negative = ((uint32_t)n ^ (uint32_t)d) >> 31;
    uint32_t un = n < 0 ? 0u - (uint32_t)n : (uint32_t)n;
    uint32_t ud = d < 0 ? 0u - (uint32_t)d : (uint32_t)d;
    uint32_t q = udivmodsi4(un, ud, NULL);
    return negative ? -(int)q : (int)q;
}

int __modsi3(int n, int d)
{
    uint32_t un = n < 0 ? 0u - (uint32_t)n : (uint32_t)n;
    uint32_t ud = d < 0 ? 0u - (uint32_t)d : (uint32_t)d;
    uint32_t r = 0;
    udivmodsi4(un, ud, &r);
    return n < 0 ? -(int)r : (int)r;
}
